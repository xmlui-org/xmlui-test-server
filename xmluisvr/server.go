package xmluisvr

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/apipkg"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/dbpkg"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/localsvr"
)

// InboundProxyProtocol defines the protocol used for inbound proxy requests.
const InboundProxyProtocol = "http"

// API is a placeholder type for API functionality.
// TODO: This appears to be unused and may need removal or proper documentation.
type API struct{}

// Server represents the main HTTP server instance with all its dependencies.
// It combines database access, API configuration, HTTP routing, and logging
// into a single cohesive server implementation.
//
// The server provides these main functionalities:
//   - Static file serving from the current directory
//   - Database query execution via /query endpoint
//   - HTTP proxy functionality via /proxy endpoint
//   - Configurable API endpoints based on JSON configuration
//   - CORS middleware for cross-origin requests
type Server struct {
	Database             dbpkg.Database      // Database connection and operations
	API                  *apipkg.API         // API configuration and handlers
	Options              *localsvr.Options   // Server configuration options
	port                 localsvr.ServerPort // HTTP server port
	SourceFile           dt.Filepath         // Path to server configuration file
	Mux                  *http.ServeMux      // HTTP request multiplexer
	cliutil.WriterLogger                     // Embedded logging functionality
}

// ServerArgs contains all the dependencies and configuration needed to create a Server.
type ServerArgs struct {
	Database   dbpkg.Database      // Database connection
	API        *apipkg.API         // API configuration
	Port       localsvr.ServerPort // HTTP server port
	SourceFile dt.Filepath         // Configuration file path
	Options    *localsvr.Options   // Server options
	Writer     CLIWriter           // CLI output writer
	Logger     *slog.Logger        // Structured logger
}

// NewServer creates a new Server instance with the provided configuration.
// If no port is specified, it defaults to localsvr.DefaultServerPort.
// The server is created with an HTTP multiplexer and embedded logging.
func NewServer(args ServerArgs) *Server {
	if args.Port == 0 {
		args.Port = localsvr.DefaultServerPort
	}
	return &Server{
		Database:     args.Database,
		API:          args.API,
		port:         args.Port,
		Options:      args.Options,
		SourceFile:   args.SourceFile,
		Mux:          http.NewServeMux(),
		WriterLogger: cliutil.NewWriterLogger(args.Writer, args.Logger),
	}
}

// Initialize prepares the server for operation by initializing the API,
// setting up HTTP routes, and opening the database connection.
// This method must be called before ListenAndServe().
func (svr *Server) Initialize(ctx Context) (err error) {
	svr.V2().InfoPrint("Initializing server")
	err = svr.API.Initialize(ctx)
	if errors.Is(err, localsvr.ErrNoAPIProvided) {
		svr.Printf("No APIConfig loaded")
		err = nil
	}
	if err != nil {
		err = fmt.Errorf("failed to load APIConfig: %w", err)
		goto end
	}

	// Add URL routes
	svr.addRoutes()

	err = svr.Database.Open(ctx)
	if err != nil {
		//err = svr.ErrorError("Failed to open database", "database_type", svr.Database.Type(), "error", err)
		goto end
	}

	svr.V2().InfoPrint("Server initialized")
end:
	return err
}

// ListenAndServe starts the HTTP server and begins listening for requests.
// The server listens on the configured port and applies CORS middleware
// to all requests. This method blocks until the server shuts down or an error occurs.
func (svr *Server) ListenAndServe(_ Context) (err error) {
	svr.InfoLoud("Server listening", "on", svr.displayHost())
	return http.ListenAndServe(svr.Host(), svr.corsMiddleware(svr.Mux))
}

// corsMiddleware applies CORS headers to all HTTP responses to enable
// cross-origin requests from web browsers. It handles preflight OPTIONS
// requests and sets permissive CORS headers.
func (svr *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Host returns the full host:port string for the server.
func (svr *Server) Host() string {
	return fmt.Sprintf("%s:%d", localsvr.DefaultServerHost, svr.port)
}

// Port returns the server's configured port number.
func (svr *Server) Port() localsvr.ServerPort {
	return svr.port
}

// addRoutes configures all HTTP routes for the server including:
//   - API endpoints (if configured)
//   - Proxy endpoint (/proxy/)
//   - Query endpoint (/query)
//   - Static file serving (/)
func (svr *Server) addRoutes() {
	svr.V2().InfoPrint("Adding HTTP server routes")
	// Handle APIConfig routes first (to match /apiFile/* before static files)
	if svr.API != nil {
		apiBasePath := string(svr.API.BasePath)
		if !strings.HasSuffix(apiBasePath, "/") {
			apiBasePath += "/"
		}
		route := fmt.Sprintf("GET  %s", apiBasePath)
		svr.V3().Printf("  — %s\n", route)
		svr.Mux.HandleFunc(route, svr.API.HandleAPIFunc(svr.Database))
	}

	// Handle proxy next
	svr.V3().Printf("  — ANY  /proxy/\n")
	for _, method := range localsvr.HTTPMethods {
		svr.Mux.HandleFunc(fmt.Sprintf("%s /proxy/", method), svr.handleProxyFunc())
	}

	// Health check endpoint (bypasses API routing for test readiness checks)
	route := "GET /healthz"
	svr.V3().Printf("  — %s\n", route)
	svr.Mux.HandleFunc(route, svr.handleHealthCheckFunc())

	// Then handle query endpoint
	route = "POST /query"
	svr.V3().Printf("  — %s\n", route)
	svr.Mux.HandleFunc(route, svr.handleQueryFunc(svr.Database))

	route = "GET  /"
	svr.V3().Printf("  — %s\n", route)
	svr.Mux.HandleFunc(route, svr.handleRootFunc())

	svr.V3().InfoPrint("HTTP server routes added")

}

// writeErrorf allows calling svr.Writer.Errorf() where it is obvious that this
// is writing to the stdio without the linter complaining that I can remove
// .Writer if called directly.
func (svr *Server) writeErrorf(format string, args ...any) {
	svr.Errorf(format, args...)
}

// logError allows calling svr.Logger.Error() where it is obvious that this
// is writing to the stdio without the linter complaining that I can remove
// .Writer if called directly.
func (svr *Server) logError(msg string, attrs ...any) {
	svr.Error(msg, attrs...)
}
