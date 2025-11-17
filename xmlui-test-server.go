package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"           // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// ===== Auth / Policy =====

const roleReader = "reader"
const roleWriter = "writer"

// ===== User Management =====
type User struct {
	Sub       string    `json:"sub"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

type UserStore struct {
	Users map[string]*User `json:"users"`
	mu    sync.RWMutex
}

type Session struct {
	Sub       string    `json:"sub"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"exp"`
}

func loadUsers(filePath string) (*UserStore, error) {
	store := &UserStore{Users: make(map[string]*User)}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Users file %s doesn't exist, starting with empty user store", filePath)
		return store, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("failed to parse users file: %w", err)
	}

	log.Printf("Loaded %d users from %s", len(store.Users), filePath)
	return store, nil
}

func (us *UserStore) save(filePath string) error {
	us.mu.RLock()
	defer us.mu.RUnlock()

	data, err := json.MarshalIndent(us, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

func (us *UserStore) lookupUser(sub, email string) (*User, error) {
	us.mu.Lock()
	defer us.mu.Unlock()

	if user, exists := us.Users[sub]; exists {
		user.LastLogin = time.Now()
		return user, nil
	}

	if email != "" {
		for _, user := range us.Users {
			if strings.EqualFold(user.Email, email) {
				oldKey := user.Sub // <-- define before delete
				delete(us.Users, user.Sub)
				user.Sub = sub
				user.LastLogin = time.Now()
				us.Users[sub] = user

				auditLogin(map[string]interface{}{
					"outcome":           "user_store_update",
					"user_store_action": "rekeyed_by_email",
					"email":             email,
					"old_sub":           oldKey,
					"new_sub":           sub,
				})

				return user, nil
			}
		}
	}

	return nil, fmt.Errorf("user not found")
}

type jwksCache struct {
	keys map[string]*rsa.PublicKey
	exp  time.Time
	ttl  time.Duration
	mu   sync.Mutex
}

func (j *jwksCache) getKey(ctx context.Context, issuer, overrideJWKS, kid string) (*rsa.PublicKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	if k, ok := j.keys[kid]; ok && now.Before(j.exp) {
		return k, nil
	}
	jwksURI := overrideJWKS
	if jwksURI == "" {
		wk := issuer
		if !strings.HasSuffix(wk, "/") {
			wk += "/"
		}
		wk += ".well-known/openid-configuration"
		req, _ := http.NewRequestWithContext(ctx, "GET", wk, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("oidc discovery: %w", err)
		}
		defer resp.Body.Close()
		var conf struct {
			JWKSURI string `json:"jwks_uri"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&conf); err != nil {
			return nil, fmt.Errorf("oidc config decode: %w", err)
		}
		jwksURI = conf.JWKSURI
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", jwksURI, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwks fetch: %w", err)
	}
	defer resp.Body.Close()
	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
			Use string `json:"use"`
			Alg string `json:"alg"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("jwks decode: %w", err)
	}
	newMap := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" || k.Use != "sig" {
			continue
		}
		nb, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eb, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		e := 0
		for i := 0; i < len(eb); i++ {
			e = (e << 8) + int(eb[i])
		}
		if e == 0 {
			e = 65537
		}
		pub := &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}
		newMap[k.Kid] = pub
	}
	j.keys = newMap
	j.exp = now.Add(j.ttl)
	if k, ok := j.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("kid %s not found in JWKS", kid)
}

func (s *Server) verifyIDToken(ctx context.Context, token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed JWT")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("bad header b64")
	}
	var hdr struct{ Alg, Kid, Typ string }
	if err := json.Unmarshal(hb, &hdr); err != nil {
		return nil, fmt.Errorf("bad header json")
	}
	if hdr.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported alg")
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad payload b64")
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(pb, &claims); err != nil {
		return nil, fmt.Errorf("bad payload json")
	}
	pub, err := s.jwksCache.getKey(ctx, s.helloIssuer, s.jwksURL, hdr.Kid)
	if err != nil {
		return nil, fmt.Errorf("jwks: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("bad sig b64")
	}
	h := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig); err != nil {
		return nil, fmt.Errorf("bad signature")
	}
	if iss, _ := claims["iss"].(string); iss != s.helloIssuer {
		return nil, fmt.Errorf("iss mismatch")
	}
	okAud := false
	switch a := claims["aud"].(type) {
	case string:
		okAud = (a == s.helloClientID)
	case []interface{}:
		for _, v := range a {
			if vs, _ := v.(string); vs == s.helloClientID {
				okAud = true
				break
			}
		}
	}
	if !okAud {
		return nil, fmt.Errorf("aud mismatch")
	}
	now := time.Now().Unix()
	exp := toInt64(claims["exp"])
	nbf := toInt64(claims["nbf"])
	leeway := int64(s.tokenLeeway.Seconds())
	if exp != 0 && now > exp+leeway {
		return nil, fmt.Errorf("token expired")
	}
	if nbf != 0 && now+leeway < nbf {
		return nil, fmt.Errorf("token not yet valid")
	}
	return claims, nil
}

func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case json.Number:
		i, _ := t.Int64()
		return i
	default:
		return 0
	}
}

// ===== Session Management =====
func generateSessionSecret() []byte {
	secret := make([]byte, 32)
	rand.Read(secret)
	return secret
}

func (s *Server) createSessionCookie(session *Session) (*http.Cookie, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	h := hmac.New(sha256.New, s.sessionSecret)
	h.Write(data)
	signature := h.Sum(nil)

	cookieValue := base64.URLEncoding.EncodeToString(data) + "." + base64.URLEncoding.EncodeToString(signature)

	return &http.Cookie{
		Name:     "session",
		Value:    cookieValue,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  session.ExpiresAt,
	}, nil
}

func (s *Server) verifySessionCookie(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil, fmt.Errorf("no session cookie")
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cookie format")
	}

	data, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid cookie data")
	}

	signature, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid cookie signature")
	}

	h := hmac.New(sha256.New, s.sessionSecret)
	h.Write(data)
	expectedSig := h.Sum(nil)

	if !hmac.Equal(signature, expectedSig) {
		return nil, fmt.Errorf("invalid cookie signature")
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("invalid session data")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// ===== Statement Gate (conservative) =====
var denyPattern = regexp.MustCompile(`(?is)\b(begin|commit|rollback|set|reset|lock|copy|call|do)\b|for\s+update`)

func singleStatementGate(q string) error {
	// strip /* */ block comments
	noBlock := regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(q, " ")
	// strip -- line comments
	lines := strings.Split(noBlock, "\n")
	var b strings.Builder
	for _, ln := range lines {
		if idx := strings.Index(ln, "--"); idx >= 0 {
			ln = ln[:idx]
		}
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	s := strings.TrimSpace(b.String())
	// count semicolons outside single-quoted strings
	inStr := false
	semi := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\'' {
			inStr = !inStr
		}
		if ch == ';' && !inStr {
			semi++
		}
	}
	// Allow 0 semicolons, or exactly 1 if it's terminal
	if semi > 1 {
		return fmt.Errorf("multiple statements not allowed")
	}
	if semi == 1 && !strings.HasSuffix(strings.TrimSpace(s), ";") {
		return fmt.Errorf("semicolon not at end: multiple statements not allowed")
	}
	if denyPattern.MatchString(s) {
		return fmt.Errorf("disallowed SQL construct for reader")
	}
	return nil
}

// ===== Simple Session-Based Authentication =====
func (s *Server) authenticateRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	session, err := s.verifySessionCookie(r)
	if err != nil {
		sendErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return "", false
	}
	return session.Role, true
}

// ===== Data Structures =====

type QueryRequest struct {
	SQL    string        `json:"sql"`
	Params []interface{} `json:"params"`
}

type LoginRequest struct {
	IDToken string `json:"id_token"`
}

type APIDescription struct {
	APIVersion  string               `json:"apiVersion"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	BasePath    string               `json:"basePath"`
	Endpoints   []EndpointDefinition `json:"endpoints"`
}

type EndpointDefinition struct {
	Path    string                      `json:"path"`
	Methods map[string]MethodDefinition `json:"methods"`
}

type MethodDefinition struct {
	Description string   `json:"description"`
	SQL         string   `json:"sql,omitempty"`
	SQLFile     string   `json:"sqlFile,omitempty"`
	Params      []string `json:"params,omitempty"`
}

type Server struct {
	db            *sql.DB
	apiDesc       *APIDescription
	apiDescPath   string
	pathRegexps   map[string]*regexp.Regexp
	showResponses bool
	dbType        string

	// Simplified auth config
	userStore     *UserStore
	usersFile     string
	sessionSecret []byte
	sessionTTL    time.Duration

	// Minimal OIDC config (only for login)
	helloIssuer   string
	helloClientID string
	jwksURL       string
	tokenLeeway   time.Duration
	jwksCache     *jwksCache
	trustedDomain string
	trustedRole   string
	trustedSub    string

	mu            sync.Mutex
}

// ===== Server Initialization =====

func NewServer(dbPath string, pgConnStr string, extensionPath string, apiDescPath string, showResponses bool, usersFile string) (*Server, error) {
	var db *sql.DB
	var err error
	var dbType string

	// Determine which database to use
	if pgConnStr != "" {
		// Use PostgreSQL if pgConnStr is provided
		log.Println("Using PostgreSQL database")
		db, err = sql.Open("postgres", pgConnStr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
		dbType = "postgres"
	} else {
		// Default to SQLite
		log.Println("Using SQLite database")
		// Simple connection string with extension loading enabled
		db, err = sql.Open("sqlite3", dbPath+"?_allow_load_extension=1")
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
		}
		dbType = "sqlite"

		// SQLite specific configurations
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		// Create memory database for extensions
		if _, err := db.Exec(`ATTACH DATABASE ':memory:' AS extension_mem`); err != nil {
			log.Printf("Failed to attach memory database: %v", err)
		}

		// Enable extension loading via PRAGMA
		if _, err := db.Exec(`PRAGMA load_extension = 1;`); err != nil {
			log.Printf("Warning: PRAGMA load_extension failed: %v", err)
		}

		// If extension is provided, try to load it
		if extensionPath != "" {
			var mu sync.Mutex
			mu.Lock()
			defer mu.Unlock()
			// Get the absolute path to the extension file
			absPath, err := filepath.Abs(extensionPath)
			if err != nil {
				log.Printf("Warning: failed to get absolute path: %v", err)
				absPath = "./" + extensionPath
			}

			// Ensure file has execute permissions (required for Linux)
			if err := os.Chmod(absPath, 0755); err != nil {
				log.Printf("Warning: failed to set execute permissions on extension: %v", err)
			}

			// Log extension loading attempt
			log.Printf("Trying to load extension: %s", absPath)

			loadQuery := fmt.Sprintf("SELECT load_extension('%s')", strings.ReplaceAll(absPath, "'", "''"))
			if _, err := db.Exec(loadQuery); err != nil {
				log.Printf("Extension loading failed with %v", err)
			} else {
				log.Println("Extension loaded successfully")
			}
		}
	}

	// Load users
	userStore, err := loadUsers(usersFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}

	// Initialize the server
	server := &Server{
		db:            db,
		pathRegexps:   make(map[string]*regexp.Regexp),
		showResponses: showResponses,
		dbType:        dbType,
		apiDescPath:   apiDescPath,
		userStore:     userStore,
		usersFile:     usersFile,
		sessionSecret: generateSessionSecret(),
		sessionTTL:    24 * time.Hour,
		mu:            sync.Mutex{},
	}

	// Load the API description if provided
	if apiDescPath != "" {
		if _, err := os.Stat(apiDescPath); os.IsNotExist(err) {
			log.Printf("API description file not found: %s", apiDescPath)
		} else {
			apiDesc, err := loadAPIDescription(apiDescPath)
			if err != nil {
				log.Printf("Warning: Failed to load API description: %v", err)
			} else {
				server.apiDesc = &apiDesc
				log.Printf("API description loaded successfully: %s (v%s)", apiDesc.Name, apiDesc.APIVersion)

				// Precompile the path regexps for faster matching
				for _, endpoint := range apiDesc.Endpoints {
					pathRegexp := pathToRegexp(endpoint.Path)
					server.pathRegexps[endpoint.Path] = regexp.MustCompile(pathRegexp)
				}
			}
		}
	}

	return server, nil
}

// ===== API Description Handling =====

// Load API description from file
func loadAPIDescription(filePath string) (APIDescription, error) {
	var apiDesc APIDescription
	data, err := os.ReadFile(filePath)
	if err != nil {
		return apiDesc, fmt.Errorf("failed to read API description file: %w", err)
	}

	err = json.Unmarshal(data, &apiDesc)
	if err != nil {
		return apiDesc, fmt.Errorf("failed to parse API description JSON: %w", err)
	}

	return apiDesc, nil
}

// Convert a path template to a regexp
// Example: "/clients/:id" -> "^/clients/([^/]+)$"
func pathToRegexp(path string) string {
	// Escape any special regexp characters in the path
	escaped := regexp.QuoteMeta(path)

	// Replace :paramName with a capturing group
	re := regexp.MustCompile(`:([^/]+)`)
	regexpPath := re.ReplaceAllString(escaped, "([^/]+)")

	// Add start and end anchors
	return fmt.Sprintf("^%s$", regexpPath)
}

// Extract path parameters from a URL based on the endpoint path template
// Example: extractPathParams("/clients/123", "/clients/:id") -> {"id": "123"}
func extractPathParams(requestPath string, endpointPath string, re *regexp.Regexp) map[string]string {
	params := make(map[string]string)

	// Extract param names from the path template
	paramNames := make([]string, 0)
	pathParts := strings.Split(endpointPath, "/")
	for _, part := range pathParts {
		if strings.HasPrefix(part, ":") {
			paramNames = append(paramNames, part[1:])
		}
	}

	// Extract values using regexp
	matches := re.FindStringSubmatch(requestPath)
	if len(matches) > 1 {
		// First match is the whole string, subsequent matches are capture groups
		for i, name := range paramNames {
			if i+1 < len(matches) {
				params[name] = matches[i+1]
			}
		}
	}

	return params
}

// Find the matching endpoint for a request path
func (s *Server) findMatchingEndpoint(requestPath string) (*EndpointDefinition, map[string]string) {
	if s.apiDesc == nil {
		return nil, nil
	}

	// Strip base path if present
	basePath := s.apiDesc.BasePath
	if basePath != "" && strings.HasPrefix(requestPath, basePath) {
		requestPath = strings.TrimPrefix(requestPath, basePath)
		if requestPath == "" {
			requestPath = "/"
		}
	}

	// Normalize the path by removing trailing slashes
	normalizedPath := strings.TrimSuffix(requestPath, "/")
	if normalizedPath == "" {
		normalizedPath = "/"
	}

	// First try exact match with normalized path
	for _, endpoint := range s.apiDesc.Endpoints {
		re, exists := s.pathRegexps[endpoint.Path]
		if !exists {
			// This shouldn't happen as we precompile all regexps
			log.Printf("Warning: No regexp for path %s", endpoint.Path)
			continue
		}

		if re.MatchString(normalizedPath) {
			params := extractPathParams(normalizedPath, endpoint.Path, re)
			return &endpoint, params
		}
	}

	// If we reach here, try matching with the original path as a fallback
	if normalizedPath != requestPath {
		for _, endpoint := range s.apiDesc.Endpoints {
			re, exists := s.pathRegexps[endpoint.Path]
			if !exists {
				continue
			}

			if re.MatchString(requestPath) {
				params := extractPathParams(requestPath, endpoint.Path, re)
				return &endpoint, params
			}
		}
	}

	return nil, nil
}

// ===== Parameter Extraction =====

// Extract query parameters from request URL
func extractQueryParams(r *http.Request) map[string]string {
	queryParams := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}
	return queryParams
}

// Extract JSON body parameters from request
func extractBodyParams(r *http.Request) (map[string]interface{}, error) {
	bodyParams := make(map[string]interface{})

	if r.Body == nil {
		return bodyParams, nil
	}

	var bodyBuffer bytes.Buffer
	bodyReader := io.TeeReader(r.Body, &bodyBuffer)

	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return bodyParams, err
	}

	if len(bodyBytes) > 0 {
		err = json.Unmarshal(bodyBytes, &bodyParams)
		if err != nil {
			return bodyParams, err
		}
	}

	// Reset r.Body for potential future use
	r.Body = io.NopCloser(&bodyBuffer)

	return bodyParams, nil
}

// ===== SQL Execution =====

// ===== SQL Execution =====

// Execute SQL query and return results as maps
func (s *Server) executeQuery(ctx context.Context, role string, sqlQuery string, params []interface{}) ([]map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Log the SQL query (just once)
	log.Printf("SQL: %s", sqlQuery)

	// Readers: enforce single-statement + disallow session/tx mutators
	if role == roleReader {
		if err := singleStatementGate(sqlQuery); err != nil {
			return nil, fmt.Errorf("reader policy: %w", err)
		}
	}

	// Handle PostgreSQL parameter placeholders ($1, $2, etc.) vs SQLite (?, ?, etc.)
	if s.dbType == "postgres" {
		for i := 1; i <= len(params); i++ {
			sqlQuery = strings.Replace(sqlQuery, "?", fmt.Sprintf("$%d", i), 1)
		}
	}

	var rows *sql.Rows
	var err error

	// Execute with read-only enforcement for readers

	if role == roleReader {
		switch s.dbType {
		case "postgres":
			tx, txErr := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
			if txErr != nil {
				return nil, txErr
			}
			defer tx.Rollback()

			rows, err = tx.QueryContext(ctx, sqlQuery, params...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			// Build result set inside the transaction
			columns, err := rows.Columns()
			if err != nil {
				return nil, err
			}
			var result []map[string]interface{}
			for rows.Next() {
				values := make([]interface{}, len(columns))
				ptrs := make([]interface{}, len(columns))
				for i := range columns {
					ptrs[i] = &values[i]
				}
				if err := rows.Scan(ptrs...); err != nil {
					return nil, err
				}
				row := make(map[string]interface{}, len(columns))
				for i, col := range columns {
					if b, ok := values[i].([]byte); ok {
						row[col] = string(b)
					} else {
						row[col] = values[i]
					}
				}
				result = append(result, row)
			}
			if err := rows.Err(); err != nil {
				return nil, err
			}
			if err := tx.Commit(); err != nil {
				return nil, err
			}
			return result, nil
		case "sqlite":
			if _, err := s.db.Exec(`PRAGMA query_only=ON`); err != nil {
				return nil, fmt.Errorf("sqlite query_only: %w", err)
			}
			defer func() { _, _ = s.db.Exec(`PRAGMA query_only=OFF`) }()
			rows, err = s.db.Query(sqlQuery, params...)
		default:
			rows, err = s.db.Query(sqlQuery, params...)
		}
	} else {
		rows, err = s.db.Query(sqlQuery, params...)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Process result rows
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				entry[col] = string(b)
			} else {
				entry[col] = val
			}
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// ===== HTTP Response Handling =====

// Send JSON response with the given status code
func (s *Server) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Generate JSON response
	responseJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Log the response if enabled - this is the ONLY place where responses should be logged
	if s.showResponses {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, responseJSON, "", "  "); err != nil {
			log.Printf("Error prettifying JSON for logging: %v", err)
		} else {
			log.Printf("Response: %s", prettyJSON.String())
		}
	}

	// Send the response
	if _, err := w.Write(responseJSON); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// Send error response with the given status code
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	log.Printf("Error: %s (Status: %d)", message, statusCode)
	http.Error(w, message, statusCode)
}

func normalizeDomain(domain string) string {
	d := strings.ToLower(strings.TrimSpace(domain))
	d = strings.TrimPrefix(d, "@")
	return d
}

func (s *Server) tryAuthorizeTrustedDomain(email, iss string) (*User, bool) {
	if s.trustedDomain == "" {
		return nil, false
	}
	if !strings.EqualFold(iss, s.helloIssuer) {
		return nil, false
	}
	if email == "" {
		return nil, false
	}
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if !strings.HasSuffix(emailLower, "@"+s.trustedDomain) {
		return nil, false
	}

	s.userStore.mu.Lock()
	defer s.userStore.mu.Unlock()

	if s.userStore.Users == nil {
		s.userStore.Users = make(map[string]*User)
	}

	user, exists := s.userStore.Users[s.trustedSub]
	if !exists {
		user = &User{
			Sub:       s.trustedSub,
			Email:     fmt.Sprintf("trusted@%s", s.trustedDomain),
			Role:      s.trustedRole,
			CreatedAt: time.Now(),
		}
		s.userStore.Users[s.trustedSub] = user
	}
	user.LastLogin = time.Now()
	return user, true
}

// ===== Request Handlers =====

// Handle login requests
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("Login: %s %s", r.Method, r.URL.Path)

	if r.Method != "POST" {
		auditLogin(map[string]interface{}{
			"outcome": "login_failure", "reason": "method_not_allowed", "method": r.Method,
			"alg": "", "kid": "",
		})
		sendErrorResponse(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		auditLogin(map[string]interface{}{
			"outcome": "login_failure", "reason": "bad_request", "error": err.Error(),
			"alg": "", "kid": "",
		})
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse header pre-verify so we can log kid/alg even on failures
	var hdrAlg, hdrKid string
	if parts := strings.Split(req.IDToken, "."); len(parts) == 3 {
		if hb, err := base64.RawURLEncoding.DecodeString(parts[0]); err == nil {
			var hdr struct{ Alg, Kid, Typ string }
			if json.Unmarshal(hb, &hdr) == nil {
				hdrAlg, hdrKid = hdr.Alg, hdr.Kid
			}
		}
	}

	// Verify the ID token (heavy lifting - done once)
	claims, err := s.verifyIDToken(r.Context(), req.IDToken)
	if err != nil {
		auditLogin(map[string]interface{}{
			"outcome": "login_failure",
			"reason":  "token_verify_failed",
			"error":   err.Error(),
			"alg":     hdrAlg, "kid": hdrKid,
		})
		sendErrorResponse(w, "Invalid ID token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Extract claims once
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	iss, _ := claims["iss"].(string)
	var audVal interface{} = claims["aud"] // may be string or []any

	if sub == "" {
		auditLogin(map[string]interface{}{
			"outcome": "login_failure",
			"reason":  "missing_sub",
			"iss":     iss, "aud": audVal, "email": email, "alg": hdrAlg, "kid": hdrKid,
		})
		sendErrorResponse(w, "ID token missing sub claim", http.StatusBadRequest)
		return
	}

	// Look up user (Dick's best practice: sub first, fallback to email)
	user, err := s.userStore.lookupUser(sub, email)
	trustedGrant := false
	if err != nil {
		if fallbackUser, ok := s.tryAuthorizeTrustedDomain(email, iss); ok {
			user = fallbackUser
			trustedGrant = true
		} else {
			auditLogin(map[string]interface{}{
				"outcome": "login_failure",
				"reason":  "user_not_authorized",
				"sub":     sub, "email": email, "iss": iss, "aud": audVal, "alg": hdrAlg, "kid": hdrKid,
			})
			sendErrorResponse(w, "User not authorized", http.StatusForbidden)
			return
		}
	}

	// Save updated user store (in case we updated sub)
	if err := s.userStore.save(s.usersFile); err != nil {
		log.Printf("Warning: failed to save user store: %v", err)
		// Optional: audit as non-fatal warning
		auditLogin(map[string]interface{}{
			"outcome": "user_store_save_failed",
			"error":   err.Error(),
			"sub":     user.Sub, "email": user.Email,
		})
	}

	// Create session
	session := &Session{
		Sub:       user.Sub,
		Role:      user.Role,
     	ExpiresAt: time.Now().Add(s.sessionTTL), 
	}

	// Create signed session cookie
	cookie, err := s.createSessionCookie(session)
	if err != nil {
		auditLogin(map[string]interface{}{
			"outcome": "login_failure",
			"reason":  "session_create_failed",
			"sub":     sub, "email": email, "iss": iss, "aud": audVal, "alg": hdrAlg, "kid": hdrKid,
		})
		sendErrorResponse(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set cookie and respond
	http.SetCookie(w, cookie)

	// Success audit
	grantedBy := "user_store"
	if trustedGrant {
		grantedBy = "trusted_domain"
	}

	auditLogin(map[string]interface{}{
		"outcome":     "login_success",
		"grant":       grantedBy,
		"sub":         user.Sub,
		"email":       user.Email,
		"presented":   email,
		"role":        user.Role,
		"iss":         iss,
		"aud":         audVal,
		"alg":         hdrAlg,
		"kid":         hdrKid,
		"trusted_dom": s.trustedDomain,
	})

	s.sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"role":    user.Role,
		"sub":     user.Sub,
	}, http.StatusOK)
}

// Handle API requests based on the API description
func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	log.Printf("API: %s %s", r.Method, r.URL.Path)

	// Simple session-based authentication
	role, ok := s.authenticateRequest(w, r)
	if !ok {
		return
	}

	if s.apiDesc == nil {
		sendErrorResponse(w, "API description not loaded", http.StatusInternalServerError)
		return
	}

	// Find the matching endpoint
	endpoint, pathParams := s.findMatchingEndpoint(r.URL.Path)
	if endpoint == nil {
		http.NotFound(w, r)
		return
	}

	// Check if the method is supported
	methodDef, exists := endpoint.Methods[r.Method]
	if !exists {
		log.Printf("Method %s not allowed for endpoint %s", r.Method, endpoint.Path)
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract parameters
	queryParams := extractQueryParams(r)

	bodyParams, err := extractBodyParams(r)
	if err != nil {
		log.Printf("Warning: Failed to parse request body as JSON: %v", err)
	}

	// Prepare SQL query
	sqlQuery := ""

	// Check if SQL should be loaded from a file
	if methodDef.SQLFile != "" {
		// Determine the API description file's directory to make relative paths work
		apiDir := filepath.Dir(s.apiDescPath)

		// Build the SQL file path relative to the API description file
		sqlFilePath := filepath.Join(apiDir, methodDef.SQLFile)
		log.Printf("Loading SQL from file: %s", sqlFilePath)

		// Read the SQL file
		sqlBytes, err := os.ReadFile(sqlFilePath)
		if err != nil {
			sendErrorResponse(w, fmt.Sprintf("Failed to read SQL file: %v", err), http.StatusInternalServerError)
			return
		}

		// Use the file contents as the SQL query
		sqlQuery = string(sqlBytes)
	} else {
		// Use the inline SQL from the API definition
		sqlQuery = methodDef.SQL
	}

	// Replace named parameters with ? placeholders and build params array
	var sqlParams []interface{}

	// If we have defined params, use them in order
	if len(methodDef.Params) > 0 {
		for _, paramName := range methodDef.Params {
			// Check path params first, then query params, then body params
			if value, ok := pathParams[paramName]; ok {
				sqlParams = append(sqlParams, value)
				sqlQuery = strings.Replace(sqlQuery, ":"+paramName, "?", 1)
			} else if value, ok := queryParams[paramName]; ok {
				sqlParams = append(sqlParams, value)
				sqlQuery = strings.Replace(sqlQuery, ":"+paramName, "?", 1)
			} else if value, ok := bodyParams[paramName]; ok {
				sqlParams = append(sqlParams, value)
				sqlQuery = strings.Replace(sqlQuery, ":"+paramName, "?", 1)
			} else {
				// Parameter not found, add nil
				sqlParams = append(sqlParams, nil)
				sqlQuery = strings.Replace(sqlQuery, ":"+paramName, "?", 1)
			}
		}
	}

	// Execute the query
	result, err := s.executeQuery(r.Context(), role, sqlQuery, sqlParams)
	if err != nil {
		sendErrorResponse(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	s.sendJSONResponse(w, result, http.StatusOK)
}

// Handle direct SQL query requests
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	log.Printf("Query: %s", r.URL.Path)

	// Simple session-based authentication
	role, ok := s.authenticateRequest(w, r)
	if !ok {
		return
	}

	if r.Method != "POST" {
		sendErrorResponse(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use io.TeeReader to log the body while still allowing it to be read
	var bodyBuffer bytes.Buffer
	teeReader := io.TeeReader(r.Body, &bodyBuffer)

	// Read the body into a buffer
	_, err := io.ReadAll(teeReader)
	if err != nil {
		sendErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Decode the body into the QueryRequest struct
	var req QueryRequest
	if err := json.NewDecoder(&bodyBuffer).Decode(&req); err != nil {
		sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Execute the query
	result, err := s.executeQuery(r.Context(), role, req.SQL, req.Params)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	s.sendJSONResponse(w, result, http.StatusOK)
}

// Handle proxy requests
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	// 1. Parse off the part after "/proxy/".
	targetPath := strings.TrimPrefix(r.URL.Path, "/proxy/")
	targetQuery := r.URL.RawQuery

	// 2. Split off the first segment as the actual host.
	pathParts := strings.SplitN(targetPath, "/", 2)
	hostPart := pathParts[0]

	// 3. The remainder is your path on that host.
	var subPath string
	if len(pathParts) > 1 {
		subPath = "/" + pathParts[1]
	} else {
		subPath = "/"
	}

	// 4. Construct a "bare" target with no path so the default Director won't double up paths.
	rawTarget := "https://" + hostPart
	targetURL, err := url.Parse(rawTarget)
	if err != nil {
		sendErrorResponse(w, "Invalid target URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 5. Create the reverse proxy.
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 6. Update the inbound request with subPath and query
	r.URL.Scheme = targetURL.Scheme
	r.URL.Host = targetURL.Host
	r.URL.Path = subPath
	r.URL.RawQuery = targetQuery

	// 7. (Optional) Reassign the Host header to match target
	r.Host = targetURL.Host

	// 8. Finally, run the proxy
	proxy.ServeHTTP(w, r)
}

func launchBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // Unix-like
		cmd = "xdg-open"
		args = []string{url}
	}

	err := exec.Command(cmd, args...).Start()
	if err != nil {
		log.Printf("Failed to launch browser: %v", err)
	}
}

func auditLogin(fields map[string]interface{}) {
	b, _ := json.Marshal(fields)

	// Append to server.log
	f, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// fallback to standard log if file can't be opened
		log.Printf("audit_fallback %s", string(b))
		return
	}
	defer f.Close()

	fmt.Fprintln(f, string(b))
}


// ===== Main Application =====

// injectPgPort injects or overrides the port in a Postgres connection string (URL or DSN format)
func injectPgPort(pgConnStr, pgPort string) string {
	if pgConnStr == "" || pgPort == "" {
		return pgConnStr
	}
	if strings.HasPrefix(pgConnStr, "postgres://") || strings.HasPrefix(pgConnStr, "postgresql://") {
		u, err := url.Parse(pgConnStr)
		if err == nil {
			if u.Port() == "" || u.Port() != pgPort {
				u.Host = u.Hostname() + ":" + pgPort
				return u.String()
			}
		}
		return pgConnStr
	}
	// DSN format: add or replace port=...
	re := regexp.MustCompile(`port=\\d+`)
	if re.MatchString(pgConnStr) {
		return re.ReplaceAllString(pgConnStr, "port="+pgPort)
	}
	return pgConnStr + " port=" + pgPort
}

func main() {
	// Set custom flag usage to display double dashes for word options
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.VisitAll(func(f *flag.Flag) {
			prefix := "-"
			// Use double dash for multi-character flags
			if len(f.Name) > 1 {
				prefix = "--"
			}
			fmt.Fprintf(flag.CommandLine.Output(), "  %s%s: %s\n", prefix, f.Name, f.Usage)
		})
	}

	// Set up command line flags with long and short versions
	var portValue string
	flag.StringVar(&portValue, "port", "8080", "Port to run the server on")
	flag.StringVar(&portValue, "p", "8080", "Port to run the server on (shorthand)")
	extension := flag.String("extension", "", "Path to SQLite extension to load")
	apiDesc := flag.String("api", "", "Path to API description file")
	dbPath := flag.String("db", "data.db", "Path to SQLite database file")
	showResponses := flag.Bool("show-responses", false, "Enable logging of SQL query responses")
	pgConnStr := flag.String("pg-conn", "", "PostgreSQL connection string (if provided, use PostgreSQL instead of SQLite)")
	pgPort := flag.String("pg-port", "", "PostgreSQL port (optional, overrides port in --pg-conn if provided)")
	usersFile := flag.String("users-file", "users.json", "Path to users JSON file")
	helloIssuer := flag.String("hello-issuer", "https://issuer.hello.coop", "Hello OIDC issuer URL")
	helloClientID := flag.String("hello-client-id", "", "OIDC client_id (audience for ID-token fallback)")
	jwksURL := flag.String("hello-jwks-url", "", "Override JWKS URL (optional)")
	tokenLeeway := flag.Int("token-leeway-seconds", 60, "Token clock skew leeway in seconds (optional)")
	trustedDomain := flag.String("hello-trusted-domain", "", "Email domain (e.g. example.com) granted fallback access")
	trustedRole := flag.String("hello-trusted-role", roleReader, "Role assigned to trusted-domain logins")

	// Short-form alias for show-responses
	var shortShowResponses bool
	flag.BoolVar(&shortShowResponses, "s", false, "Enable logging of SQL query responses (shorthand)")

	flag.Parse()

	// Set up logging
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.Println("Server starting...")

	// Print current working directory
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Working directory: %s", pwd)

	// Initialize server
	showResponsesEnabled := *showResponses || shortShowResponses
	finalPgConnStr := injectPgPort(*pgConnStr, *pgPort)
	server, err := NewServer(*dbPath, finalPgConnStr, *extension, *apiDesc, showResponsesEnabled, *usersFile)
	if err != nil {
		log.Fatal(err)
	}

	// wire auth config
	server.helloIssuer = *helloIssuer
	server.helloClientID = *helloClientID
	server.jwksURL = *jwksURL
	server.tokenLeeway = time.Duration(*tokenLeeway) * time.Second
	if server.jwksCache == nil {
		server.jwksCache = &jwksCache{keys: map[string]*rsa.PublicKey{}, ttl: 12 * time.Hour}
	}
	if server.helloClientID == "" {
		log.Println("WARNING: --hello-client-id not set; token 'aud' will fail verification")
	}
	domain := normalizeDomain(*trustedDomain)
	server.trustedDomain = domain
	if *trustedRole != "" {
		server.trustedRole = *trustedRole
	} else {
		server.trustedRole = roleReader
	}
	if domain != "" {
		server.trustedSub = fmt.Sprintf("sub_trusted_%s", strings.ReplaceAll(domain, ".", "_"))
		log.Printf("Trusted domain enabled: %s -> subject %s (role=%s)", domain, server.trustedSub, server.trustedRole)
	}

	// Create router
	mux := http.NewServeMux()

	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Handle login endpoint
	mux.HandleFunc("/login", server.handleLogin)

	// Handle API routes first (to match /api/* before static files)
	if server.apiDesc != nil {
		apiBasePath := server.apiDesc.BasePath
		if !strings.HasSuffix(apiBasePath, "/") {
			apiBasePath += "/"
		}
		mux.HandleFunc(apiBasePath, server.handleAPI)
	}

	// Handle proxy next
	mux.HandleFunc("/proxy/", server.handleProxy)

	// Then handle query endpoint
	mux.HandleFunc("/query", server.handleQuery)

	// Handle root and static files
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request for: %s", r.URL.Path)

		if r.URL.Path == "/" {
			log.Println("Trying to serve index.html")
			http.ServeFile(w, r, "index.html")
			return
		}

		filePath := "." + r.URL.Path
		log.Printf("Trying to serve: %s", filePath)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("File not found: %s", filePath)
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filePath)
	})

	// Log server settings
	log.Printf("Server configuration:")
	log.Printf("- Port: %s", portValue)
	log.Printf("- API Description: %s", *apiDesc)
	log.Printf("- Extension: %s", *extension)
	log.Printf("- Show Responses: %v", showResponsesEnabled)
	log.Printf("- Users File: %s", *usersFile)
	log.Printf("- Auth: issuer=%s client_id=%s", server.helloIssuer, server.helloClientID)

	if *pgConnStr != "" {
		log.Printf("- Database: PostgreSQL")
	} else {
		os.Setenv("STEAMPIPE_CACHE", "false")
		log.Printf("- Database: SQLite (data.db)")
	}

	// Start server
	log.Printf("Server listening on %s...", portValue)
	if err := http.ListenAndServe("0.0.0.0:"+portValue, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
