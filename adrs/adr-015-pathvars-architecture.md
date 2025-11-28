# ADR-015: PathVars Package Architecture & Design

- **Status**: Active
- **Date**: 2025-10-15
- **Authors**: Mike Schinkel, Claude (Anthropic)
- **Related ADRs**: ADR-009 (RFC 9457 Error Handling), ADR-010 (Error Handling Implementation)

## Executive Summary

PathVars is a URL path template and routing package for a local development XML UI server. It enables dynamic API endpoint configuration through JSON files, where endpoints are defined with path templates containing typed, validated parameters. The package focuses on developer-friendly error messages and maintainability over performance, as it targets local development environments.

## Problem Statement

The XML UI Server needs to:
1. Load API endpoint configurations from JSON files at startup
2. Match incoming HTTP requests to configured endpoints
3. Extract and validate parameters from URL paths
4. Provide clear error messages to developers when validation fails
5. Support SQL query execution with extracted parameters

Current configuration format:
```json
{
  "endpoints": [
    {
      "path": "GET /users/{id:int}/posts/{slug:slug:length[5..50]}",
      "sql": "SELECT * FROM posts WHERE user_id = :id AND slug = :slug"
    },
    {
      "path": "GET /archive/{post_date*:date:format[yyyy/mm/dd]}",
      "sql": "SELECT * FROM posts WHERE DATE(created) >= :post_date"
    }
  ]
}
```

## Core Requirements

### Functional Requirements
1. **Parse path templates** with syntax: `{name}`, `{name:type}`, `{name:type:constraint}`, `{name*:type:constraint}` (multi-segment)
2. **Support HTTP method** in path specification: `GET /path`, `POST /path`, or just `/path` (any method)
3. **Extract parameters** from incoming request paths, including multi-segment parameters
4. **Validate parameters** against types and constraints
5. **Pre-compile regexs** at server startup for efficiency and performance
6. **Return endpoint index** to caller for looking up SQL queries

### Non-Functional Requirements
1. **Developer-friendly errors** - Clear messages for less technical users
2. **Maintainability** over performance - This is for local dev, not production
3. **Memory efficiency** - Dev server may run for days/weeks continuously
4. **Extensibility** - Easy to add new types and constraints

## Recent Architectural Improvements

### Constraint System Redesign (Latest Version)

**Problem**: The original constraint system had architectural issues:
- Special case handling for range constraints
- Silently ignored parse errors
- Proxy patterns that violated self-contained design
- State machine logic mixed with constraint-specific parsing

**Solution**: Complete redesign of the constraint parsing system:

1. **Self-Contained Constraints**: Each constraint type now handles its own parsing with full context
   - `Parse(value string, dataType PVDataType) (Constraint, error)` interface
   - Range constraints automatically select appropriate implementation (int, decimal, date) based on data type
   - No more special cases in the main parsing logic

2. **Proper Error Propagation**:
   - Parse errors are collected and returned immediately
   - Unknown constraints cause startup failures, not runtime surprises
   - Clear, descriptive error messages with context

3. **Clean State Machine**:
   - Constraint parsing operates purely on syntax, not constraint identity
   - Character validation only applies in appropriate parsing modes
   - No special cases for specific constraint types

4. **Consistent Syntax**:
   - All format constraints now use `format[...]` syntax
   - Date formats require explicit constraint syntax: `{date:date:format[yyyy-mm-dd]}`
   - No implicit format strings without constraint wrapper

**Benefits**:
- **Maintainability**: Adding new constraints requires no changes to core parsing logic
- **Reliability**: Invalid configurations fail immediately at startup
- **Clarity**: Error messages clearly indicate what went wrong and where
- **Extensibility**: Constraint system follows "closed for modification, open for extension" principle

### Breaking Changes in Latest Version

**Date Format Syntax**: Date format constraints now require explicit `format[...]` wrapper:
```
// OLD (no longer supported)
{date:date:yyyy-mm-dd}
{post_date*:date:yyyy/mm/dd}

// NEW (required)
{date:date:format[yyyy-mm-dd]}
{post_date*:date:format[yyyy/mm/dd]}
```

**Error Behavior**: Unknown constraints now cause startup errors instead of being silently ignored:
```
// OLD: Would be ignored silently
{value:string:unknown[param]}

// NEW: Causes immediate error with clear message
Error: unknown constraint type
constraint_spec=unknown[param]
constraint_type=unknown
```

**Constraint Interface**: If you've implemented custom constraints, update the `Parse` method signature:
```go
// OLD
Parse(value string) (Constraint, error)

// NEW
Parse(value string, dataType PVDataType) (Constraint, error)
```

These changes improve reliability and consistency but require updating existing configurations to use the new syntax.

## Architecture Decisions

### 1. Package Name: `pathvars`
- **Why**: Emphasizes variable substitution and extraction, not just parsing
- **Rejected alternatives**:
    - `pathparser` - Too focused on parsing
    - `urltemplate` - Too generic, might conflict
    - `pathmatch` - Emphasizes matching over substitution

### 2. Extended URI Template Syntax
- **Decision**: Use `{name:type:constraint}` and `{name*:type:constraint}` syntax for path parameters and query parameters, with implicit type inference
- **Why**:
    - Inline definitions are clearer for non-technical users
    - Everything about a parameter is visible in one place
    - No need to cross-reference separate schema files
    - Multi-segment support (`*` suffix) handles variable-length paths elegantly
    - Query parameter support enables complete URL template definition in one place
    - Implicit type inference reduces verbosity when parameter names match data types
- **Trade-off**: Not RFC 6570 compliant, but more practical
- **Examples**:
    - `{id:int}` - Single segment parameter with explicit type
    - `{int}` - Single segment parameter with implicit int type (inferred from name)
    - `{name:string:length[5..50]}` - Single segment with constraint
    - `{slug::enum[news,sports,tech]}` - Implicit slug type with constraint (double colon)
    - `{date*:date:format[yyyy/mm/dd]}` - Multi-segment date parameter with explicit type and format constraint
    - `{date*}` - Multi-segment date parameter with implicit type
    - `/users/{id:int}?{limit?10:int}&{offset?0:int}` - Path with query parameters

### 3. Memory Management Strategy
- **Decision**: Return `MatchResult` by value, not pointer
- **Why**: Avoid heap allocations on every request (thousands over days)
- **Decision**: Private `params` map with accessor methods
- **Why**: Allows future optimization without breaking API compatibility
- **Future options**:
    - Stack-allocated arrays for small param counts
    - Sync.Pool for map reuse
    - Custom packed storage

### 4. Error Handling Approach
- **Decision**: Sentinel errors + `errors.Join()` with metadata
- **Why**:
    - Custom error types create complexity in Go
    - Type assertions become problematic across package boundaries
    - `errors.Join()` provides rich context without custom types
- **Example**:
  ```go
  doterr.NewErr(
    ErrValidationFailed,
    "parameter", "score",
    "value", "999",
    "expected", "integer between 0 and 100"
  )
  ```

### 5. Two-Phase Processing
- **Decision**: Separate compile phase and match phase
- **Compile phase** (startup):
    - Parse all templates
    - Build regexes
    - Validate configuration
    - Pre-allocate data structures
- **Match phase** (per request):
    - Linear search through routes (simple, sufficient for dev)
    - Extract parameters
    - Return by value

### 6. Type System Design

**Core Types** (built-in validation):
- `string` - Any text (default if no type specified)
- `integer` - Integer values
- `decimal` - Decimal numbers
- `real` - Real numbers (floating point)
- `identifier` - Lowercase, starts with letter, then alphanumeric/underscore
- `date` - Date/time values
- `uuid` - Standard UUID format
- `alphanumeric` - Alphanumeric only
- `slug` - URL-safe slug format
- `boolean` - true/false

**Why these types**: Common in REST APIs, each has specific validation rules

### 7. Implicit Type Inference

**Automatic type detection** based on parameter names for cleaner syntax:

**How it works**: When a parameter name exactly matches a data type name, the type is automatically inferred, reducing verbosity in common cases.

**Supported Syntax Patterns**:
- `{name}` - If `name` matches a data type, infer that type; otherwise default to `string`
- `{name::constraint}` - Infer type from `name`, apply constraint (double colon syntax)
- `{name:type:constraint}` - Explicit type (original syntax still supported)

**Examples**:
```
{int}                    → Inferred as int type
{string}                 → Inferred as string type
{decimal}                → Inferred as decimal type
{real}                   → Inferred as real type
{date}                   → Inferred as date type
{slug::enum[a,b,c]}      → Inferred as slug type with enum constraint
{uuid}                   → Inferred as uuid type
{userId}                 → Not a type name, defaults to string type
{myCustomParam:int}      → Explicit type override
```

**Benefits**:
- **Shorter syntax** for common cases: `{int}` vs `{id:int}`
- **Self-documenting** parameter names that match their types
- **Backwards compatible** - explicit syntax still works
- **Validation consistency** - same validation rules apply regardless of syntax

**Double Colon Syntax** (`{name::constraint}`):
- Only allowed when `name` matches a valid data type
- Automatically infers the type from the parameter name
- Enables constraints without repeating the type name
- Error if parameter name doesn't match a known data type

**Inference Rules**:
1. Parameter name must exactly match a data type name (case-sensitive)
2. If no match found, defaults to `string` type (for `{name}` syntax)
3. Double colon syntax requires a type match or produces an error
4. Explicit type syntax always takes precedence over inference

### 8. Constraint System

**Constraint Categories**:
1. **Range constraints**: `range[0..100]` for numeric bounds
2. **Length constraints**: `length[5..50]` for string length
3. **Regex constraints**: `regex[regex]` for custom patterns
4. **Enum constraints**: `enum[val1,val2,val3]` for fixed values
5. **Format constraints**: Built-in aliases (`format[dateonly]`, `format[utc]`, `format[local]`, `format[datetime]`) or custom token-based formats (`format[yyyy-mm-dd]`) for date/time, and version formats (`format[v4]`, `format[ulid]`) for UUIDs
6. **Notempty constraints**: `notempty` to ensure values are not empty

**Architecture**: Self-contained constraint system with improved error handling
- Each constraint implements `Constraint` interface with `Parse(value string, dataType PVDataType) (Constraint, error)`
- **No special cases**: All constraints are parsed through the same mechanism
- **Proper error propagation**: Parse errors are collected and returned, not silently ignored
- **Type-aware parsing**: Constraints receive the data type context for intelligent parsing
- **Fail-fast validation**: Invalid constraints cause startup errors, not runtime surprises

**Design Principles**:
- **Self-contained**: Each constraint knows how to parse itself given a data type
- **No proxy patterns**: Range constraints automatically select the correct implementation based on data type
- **Clear error messages**: Unknown constraints cause immediate, descriptive errors
- **Extensible**: Adding new constraints requires only implementing the interface

#### Date and Time Format Constraints

The date type supports extensive format constraints using the `format[...]` syntax:

**Built-in Format Aliases**:
- `format[dateonly]` - Date only (yyyy-mm-dd): `2023-12-25`
- `format[utc]` - Strict UTC timestamps (yyyy-mm-ddThh:mm:ssZ, Z required): `2023-12-25T10:30:00Z`
- `format[local]` - Timezone-naive timestamps (yyyy-mm-ddThh:mm:ss, Z forbidden): `2023-12-25T10:30:00`
- `format[datetime]` - Flexible timestamps (yyyy-mm-ddThh:mm:ss with optional Z, defaults to UTC): `2023-12-25T10:30:00` or `2023-12-25T10:30:00Z`

**Choosing the Right Format**:
- Use `dateonly` for pure dates without time information (birthdays, event dates)
- Use `utc` when you need strict UTC enforcement (distributed systems, logging)
- Use `local` for timezone-naive timestamps (like SQL's `timestamp without time zone`)
- Use `datetime` for flexible APIs that accept both formats (convenience for API consumers)

**Custom Date Formats** (using token-based parsing):
- `format[yyyy-mm-dd]` - ISO date: `2023-12-25`
- `format[mm-dd-yyyy]` - US format: `12-25-2023`
- `format[dd-mm-yyyy]` - European format: `25-12-2023`

**Time Formats**:
- `format[hh:mm:ss]` - Full time: `15:30:45`
- `format[hh:mm]` - Hours and minutes: `15:30`
- `format[hh]` - Hours only: `15`

**DateTime Formats**:
- `format[yyyy-mm-dd_hh:mm:ss]` - Date and time: `2023-12-25_15:30:45`
- `format[yyyy-mm-dd_hh:mm]` - Date and hour/minute: `2023-12-25_15:30`
- `format[yyyy-mm-dd_hh]` - Date and hour: `2023-12-25_15`
- `format[dd-mm-yyyy_hh:mm:ss]` - European date/time: `25-12-2023_15:30:45`
- `format[mm-dd-yyyy_hh:mm:ss]` - US date/time: `12-25-2023_15:30:45`

**Creative Date Formats**:
PathVars supports highly customizable date format strings for specialized use cases:
- `format[my-dear-aunt-sally-was-born-on-yyyy-at-hh:mm-in-the-morning]`
- `format[the-year-yyyy-month-mm-day-dd]`
- `format[log-entry-yyyy-mm-dd-at-hh:mm:ss.log]`

This allows for natural language-like date formats while maintaining precise validation of the date/time tokens.

**MM/II Disambiguation**:

Since `mm` can mean either "month" or "minutes", PathVars uses context-aware parsing:

- `mm` alone is **ambiguous** and will be rejected
- `mm` following `hh` (hours) means **minutes**: `hh:mm`
- `mm` without preceding `hh` means **months**: `yyyy-mm-dd`
- `ii` explicitly means **minutes** when hours are not present: `mm_ii`

**Examples**:
```
{date:date:format[yyyy-mm-dd]}      ✓ mm = months (2023-12-25)
{time:date:format[hh:mm:ss]}        ✓ mm = minutes (15:30:45)
{monthmin:date:format[mm_ii]}       ✓ mm = months, ii = minutes (12_30)
{month:date:format[mm]}             ✗ Ambiguous - rejected at parse time
```

This disambiguation makes the format intuitive for laypersons while avoiding ambiguity in parameter validation.

### 9. UUID Format Constraints

**Syntax**: Use `format[...]` with UUID format specifications for strict UUID validation.

**Standard UUID Formats**:
- `format[v1]` - Time + MAC address (UUID version 1)
- `format[v2]` - Time + POSIX UID/GID (UUID version 2)
- `format[v3]` - Name-based, MD5 (UUID version 3)
- `format[v4]` - Random (UUID version 4, most common)
- `format[v5]` - Name-based, SHA-1 (UUID version 5)
- `format[v6]` - Reordered v1, time-ordered (UUID version 6)
- `format[v7]` - Unix timestamp + random (UUID version 7, modern default)
- `format[v8]` - Custom/experimental (UUID version 8)

**UUID Version Ranges**:
- `format[v1-5]` or `format[v1to5]` - Accepts UUID versions 1-5
- `format[v6-8]` or `format[v6to8]` - Accepts UUID versions 6-8
- `format[any]` or `format[generic]` - Accepts any valid UUID (versions 1-8)

**Alternative ID Formats** (use with `string` type):
- `format[ulid]` - ULID (26 chars, Crockford Base32, lexicographically sortable)
- `format[ksuid]` - KSUID (27 chars, Base62, K-sortable)
- `format[nanoid]` - NanoID (21 chars, URL-safe, short IDs)

**Examples**:
```
{id:uuid:format[v4]}           → Strict v4 UUID validation
{user_id:uuid:format[v7]}      → Modern timestamp-based UUID
{object_id:uuid:format[any]}   → Any valid UUID version
{log_id:string:format[ulid]}   → ULID for lexicographic sorting
{session:string:format[nanoid]} → Short, URL-safe session ID
```

**Implementation Notes**:
- Standard UUIDs (v1-v8) use `uuid` data type with standard 8-4-4-4-12 hex format
- Alternative formats (ULID, KSUID, NanoID) use `string` data type for flexibility
- All UUID constraints validate both format structure and version/variant bits
- Zero dependencies: Pure Go implementation without external UUID libraries

### 10. Multi-Segment Parameters

**Variable-length path parameters** for advanced use cases like date archives:

**Syntax**: Use `{name*:type:constraint}` where the `*` indicates the parameter can span multiple path segments.

**Example Use Case**: Archive URLs with variable date precision:
- `/archive/2025` (year only)
- `/archive/2025/09` (year and month)
- `/archive/2025/09/18` (full date)

**Configuration**:
```json
{
  "path": "GET /archive/{post_date*:date:format[yyyy/mm/dd]}",
  "sql": "SELECT * FROM posts WHERE date_part >= :post_date"
}
```

**How it works**:
- Multi-segment parameters capture `([^/]+(?:/[^/]+)*)` instead of `([^/]+)`
- Date constraints support partial validation (year-only, year-month, full date)
- Both constraint format and URL format use natural slashes
- Extracted parameter value: `"2025/09/18"` (maintains URL format for easy SQL usage)

**Constraints for Multi-Segment Dates**:
- Use natural slash separators: `format[yyyy/mm/dd]`, `format[mm/dd/yyyy]`, `format[dd/mm/yyyy]`
- Format matches URL structure for intuitive configuration
- Partial matching validates each segment appropriately

**Limitations**:
- Multi-segment parameters must be the last parameter in the path template
- Cannot have literal segments after a multi-segment parameter
- Use sparingly - most use cases work better with separate routes

### 10. Query Parameter Support

**URL template with query parameters** for complete API endpoint definition:

**Syntax**: Use `?{name:type:constraint}&{name2:type:constraint}` after the path portion to define query parameters.

**Default Values**: Use `{name?default_value:type:constraint}` syntax to specify default values for optional query parameters.

**Example Use Cases**:
- Pagination: `/users?{limit?10:int}&{offset?0:int}`
- Filtering: `/products?{category:string}&{min_price?0:decimal}`
- Search: `/search/{query:string}?{page?1:int}&{per_page?20:int:range[1..100]}`

**Configuration**:
```json
{
  "path": "GET /users/{id:int}?{include_posts?false:boolean}&{limit?10:int:range[1..100]}",
  "sql": "SELECT * FROM users WHERE id = :id"
}
```

**How it works**:
- Query parameters are extracted from the URL query string
- Missing query parameters use their default values if specified
- Query parameters follow the same type and constraint validation as path parameters
- Both path and query parameters are available in the MatchResult
- SQL queries can reference both path and query parameters with `:parameter_name`

**Supported Syntax**:
- `{name:type}` - Required query parameter
- `{name?default:type}` - Optional query parameter with default value
- `{name:type:constraint}` - Required query parameter with constraint
- `{name?default:type:constraint}` - Optional query parameter with default and constraint

**Examples**:
```
/api/users?{active?true:boolean}                    ✓ Optional boolean with default
/search?{q:string}&{limit?20:int:range[1..100]}  ✓ Required + optional with constraint
/posts/{id:int}?{format?json:enum[json,xml]}     ✓ Path param + optional query enum
```

**Integration with Path Parameters**:
- Path parameters are extracted first, then query parameters
- Both are available in the same MatchResult
- Parameter names must be unique across path and query parameters
- Validation applies to both path and query parameters

### 11. Router Design

**Not a general-purpose router** - Specifically for this API use case:
- Returns endpoint index, not handlers
- Supports method+path format from JSON config
- No middleware, no route groups, no complex features
- Linear search is fine (local dev, <100 routes typically)

### 12. Future Type Addition Strategy

**Design Question**: When adding new data types to PathVars, should they be included in the implicit type inference system?

**Background**: The current system allows `{name}` syntax where:
- If `name` matches an existing data type (like `{int}`, `{date}`, `{slug}`), it infers that type
- If `name` doesn't match a type, it defaults to `string`

This creates a **strategic decision** for each new data type: should `{newtype}` automatically work, or should users be required to write `{param:newtype}` explicitly?

**Options Considered**:

1. **Automatic Inclusion (Permissive)**
   - **Approach**: Add all new types to the inference system
   - **Benefits**: Consistent developer experience, `{newtype}` syntax works immediately
   - **Risks**: Breaking changes if users have parameters named after future types
   - **Example**: Adding `time` type could break existing `{time}` parameters that expect string behavior

2. **Explicit Only (Conservative)**
   - **Approach**: Never add new types to inference, require explicit syntax
   - **Benefits**: Zero breaking changes, completely predictable behavior
   - **Drawbacks**: Inconsistent UX (some types inferred, others explicit), more verbose
   - **Example**: Users must write `{param:newtype}` while `{int}` still works

3. **Frozen Inference List (Hybrid)**
   - **Approach**: Freeze the current inference list, new types always explicit
   - **Benefits**: No breaking changes, some inference still available
   - **Drawbacks**: Creates two classes of types with different syntax requirements
   - **Example**: `{int}` works but `{newtype}` doesn't, confusing for newcomers

**Decision**: **Defer until needed** - Evaluate each new data type individually when added.

**Rationale**:
- **Avoids premature decisions**: We don't know what types we'll add or how commonly they'll be used
- **Prevents inconsistent UX**: Option 3 would create hard-to-explain inconsistencies
- **Keeps all options open**: We can choose the most appropriate approach for each specific type
- **No immediate pressure**: The current type set covers the majority of REST API use cases

**Evaluation Criteria for Future Types**:
When adding a new data type, consider:
1. **Collision risk**: How likely are existing parameter names to match the type name?
2. **Usage frequency**: Will this be a common type that benefits from terse syntax?
3. **Breaking change impact**: How many existing configurations might be affected?
4. **Developer expectations**: Would users expect this type to support inference?

**Examples of Future Evaluation**:
- **High collision risk**: `time`, `date`, `user`, `id` - common parameter names, high breaking change risk
- **Low collision risk**: `ipv4`, `semver`, `jwt` - specialized names, unlikely to be used as generic parameter names
- **Common usage**: `email`, `url` - frequently used, would benefit from terse syntax
- **Specialized usage**: `base64`, `hex` - less common, explicit syntax acceptable

This approach prioritizes **stability and consistency** while keeping the door open for **pragmatic decisions** as the system evolves.

## Data Flow

### Startup Flow
```
JSON Config → Router.Add() → Parse Templates → Ready
```

1. Load endpoint configs from JSON
2. For each endpoint, call `router.Add(endpoint.Path, index)`
3. Parser extracts method, segments, parameters, types, constraints
4. Build regex for each template
5. Store compiled route with its config index

### Request Flow
```
HTTP Request → Router.Match() → MatchResult → Execute SQL
```

1. Receive HTTP request with method and path
2. Router.Match(method, path) searches routes
3. First matching route returns MatchResult with:
    - Index (for SQL lookup)
    - Extracted parameters
4. Validate parameters if needed
5. Use index to get SQL from config
6. Substitute parameters into SQL query

## Usage Example

### Configuration
```json
{
  "endpoints": [
    {
      "path": "GET /users/{id:int}",
      "sql": "SELECT * FROM users WHERE id = :id"
    },
    {
      "path": "GET /users/{id:int}/posts/{slug:slug:length[5..50]}",
      "sql": "SELECT * FROM posts WHERE user_id = :id AND slug = :slug"
    },
    {
      "path": "POST /users/{id:int}/follow/{target:int}",
      "sql": "INSERT INTO follows (user_id, target_id) VALUES (:id, :target)"
    },
    {
      "path": "GET /posts/date/{post_date:date:format[dateonly]}",
      "sql": "SELECT * FROM posts WHERE DATE(created) = :post_date"
    },
    {
      "path": "GET /events/{timestamp:date:format[utc]}",
      "sql": "SELECT * FROM events WHERE timestamp = :timestamp"
    },
    {
      "path": "GET /logs/time/{log_time:date:format[hh:mm:ss]}",
      "sql": "SELECT * FROM logs WHERE TIME(created) = :log_time"
    },
    {
      "path": "GET /schedules/{month_minute:date:format[mm_ii]}",
      "sql": "SELECT * FROM schedules WHERE month = :month AND minute = :minute"
    },
    {
      "path": "GET /users/{id:uuid:format[v4]}",
      "sql": "SELECT * FROM users WHERE uuid = :id"
    },
    {
      "path": "GET /objects/{id:uuid:format[v7]}",
      "sql": "SELECT * FROM objects WHERE id = :id"
    },
    {
      "path": "GET /logs/{id:string:format[ulid]}",
      "sql": "SELECT * FROM logs WHERE ulid = :id"
    },
    {
      "path": "GET /sessions/{token:string:format[nanoid]}",
      "sql": "SELECT * FROM sessions WHERE token = :token"
    },
    {
      "path": "GET /archive/{post_date*:date:format[yyyy/mm/dd]}",
      "sql": "SELECT * FROM posts WHERE DATE(created) >= :post_date"
    },
    {
      "path": "GET /files/{file_path*}",
      "sql": "SELECT * FROM files WHERE path LIKE :file_path"
    },
    {
      "path": "GET /users?{active?true:boolean}&{limit?20:int:range[1..100]}",
      "sql": "SELECT * FROM users WHERE active = :active LIMIT :limit"
    },
    {
      "path": "GET /products/{category:string}?{min_price?0:decimal}&{sort?name:enum[name,price,date]}",
      "sql": "SELECT * FROM products WHERE category = :category AND price >= :min_price ORDER BY :sort"
    },
    {
      "path": "GET /measurements/{value:real:range[0..1000.5]}",
      "sql": "SELECT * FROM measurements WHERE value = :value"
    },
    {
      "path": "GET /api/{uuid}/status",
      "sql": "SELECT * FROM api_objects WHERE uuid = :uuid"
    },
    {
      "path": "GET /scores/{int::range[0..100]}",
      "sql": "SELECT * FROM scores WHERE value = :int"
    },
    {
      "path": "GET /tags/{slug::enum[tech,news,sports]}",
      "sql": "SELECT * FROM articles WHERE tag = :slug"
    }
  ]
}
```

### Server Code
```go
// Startup
router := pathvars.NewRouter()
for i, ep := range config.Endpoints {
    err := router.AddRoute(ep.Method,ep.Path, i)
    if err != nil {
        log.Error("Invalid endpoint", "index", i, "error", err)
        return err
    }
}

// Per Request
result, err := router.Match(req.Method, req.URL.Path)
if err != nil {
    if errors.Is(err, pathvars.ErrNoMatch) {
        // Try 404 handler
        result, _ = router.Match("", "/404")
    }
}

// Get SQL and parameters
endpoint := config.Endpoints[result.Index]
sql := endpoint.SQL

// Substitute parameters (pseudocode)
if result.HasParams() {
    result.ForEachParam(func(name, value string) bool {
        sql = strings.ReplaceAll(sql, ":"+name, value)
        return true
    })
}
```

## Key Design Principles

### 1. Developer Experience First
- Clear error messages over terse ones
- Obvious API over clever abstractions
- Configuration in one place (inline with path)

### 2. Appropriate Engineering
- Pre-compile because it's obviously correct
- Don't over-optimize for a local dev server
- But don't be wasteful with memory on long-running process

### 3. Extensibility Without Complexity
- New types: Add to DataType enum and validator
- New constraints: Implement Constraint interface
- Future optimization: Hidden behind MatchResult accessors

### 4. Fail Fast and Clear
- Validate everything at startup when possible
- Runtime errors include context for debugging
- No silent failures or unclear states

## What PathVars Is NOT

1. **Not RFC 6570 compliant** - We extend the syntax for practical reasons
2. **Not a general router** - Specifically for SQL endpoint mapping
3. **Not optimized for scale** - Linear search is fine for dev usage
4. **Not a validation library** - Just enough validation for path parameters
5. **Not thread-safe** - Single-threaded dev server context

## Success Criteria

1. **Developer can understand errors** without reading source code
2. **Adding new parameter types** requires minimal code changes
3. **Memory usage remains flat** over days of operation
4. **Configuration is self-documenting** (no separate schema files)
5. **Server startup fails clearly** if configuration is invalid

## Future Considerations

### Possible Extensions (Future Versions)
- Parameter transformation (uppercase, lowercase, trim)
- Optional parameters with defaults
- ✅ Multi-segment parameters (`{path*}`) - **IMPLEMENTED**
- ✅ Query string parameter extraction - **IMPLEMENTED**
- ✅ Implicit type inference (`{int}`, `{slug::constraint}`) - **IMPLEMENTED**
- ✅ Self-contained constraint system - **IMPLEMENTED**
- ✅ Proper error propagation for constraints - **IMPLEMENTED**
- ✅ Creative date format support - **IMPLEMENTED**
- ✅ UUID format constraints (v1-v8, ULID, KSUID, NanoID) - **IMPLEMENTED**
- Custom validator registration
- Multi-segment parameters in middle of path (currently only supported at end)

### Performance Optimizations (If Ever Needed)
- Trie-based routing for prefix matching
- Compiled state machine instead of regex
- Parameter value caching
- JIT compilation of SQL with parameters

These are documented to show the design leaves room for growth without requiring major refactoring.

## Summary

PathVars is a pragmatic solution for a specific problem: enabling dynamic API configuration in a local development server. It prioritizes developer experience and maintainability while following obvious best practices like pre-compilation. The design carefully balances simplicity with extensibility, making trade-offs appropriate for its use case rather than trying to be a general-purpose solution.