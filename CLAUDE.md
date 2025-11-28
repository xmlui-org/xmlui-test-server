# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## File Organization for Claude-Generated Files

**CRITICAL**: All markdown files, specifications, notes, session tracking, and documentation that Claude generates should be written to:

**`./~/`** (tilde-slash directory at project root)

This directory:
- Sorts last alphabetically in the IDE (after all letter-named directories)
- Is `.gitignore`d to prevent accidental commits
- Contains only temporary/generated files

**File Naming Conventions**:
- Specifications: `~/SPEC_*.md` or `~/*_SPEC.md`
- Completion reports: `~/COMPLETED_*.md` or `~/*_COMPLETED.md`
- Session notes: `~/SESSION_NOTES.md`
- Task tracking: `~/TODO.md`, `~/TASKS_*.md`
- Technical debt: `~/TECHNICAL_DEBT.md`

**Never write generated markdown files to**:
- ❌ Project root directory
- ❌ User's home directory (`~`)
- ❌ Any committed source directories

## Project Overview

xmlui-test-server is a lightweight Go HTTP server that provides:
- Static file serving from the current directory  
- `/query` endpoint for executing SQL against SQLite or PostgreSQL databases
- `/proxy` endpoint for proxying requests to external APIs (CORS bypass)
- Optional SQLite extension loading (e.g., Steampipe plugins)
- API endpoint routing based on JSON configuration files

## Build Commands

### Standard Build
```bash
make build
```

## Running tests
```bash
make test 
make test xmluisvrr test 
```

## Running the Server

Basic usage:
```bash
make run
```

Common options:
```bash
TODO: NEEDS TO UPDATE Makefile to support parameters
TODO: THEN THIS NEEDS TO BE UPDATED WITH EXAMPLES
make run ...
```

## Architecture

### Core Components

**Server struct** (`Server`): Central handler containing database connection, API description, path regexes cache, and configuration flags.

**Database abstraction**: Supports both SQLite (default) with plans for DuckDB, PostgreSQL, MySQL etc.  with automatic parameter placeholder conversion (e.g. `?` → `$1`, `$2` for PostgreSQL).

**API Description system**: JSON-based configuration defining endpoints with path and query params (`{name:type:comma_sep_constraints}`), HTTP methods, and SQL queries. Supports both inline SQL and external SQL files via `query_file` property.

**Parameter extraction**: Unified parameter handling from URL path segments (`{param}`), query parameters, and JSON request bodies (`{topLevelProp.propsList.0.subProp`).

### Request Flow

1. **API requests** (`/api/*`): Route through API description matching, parameter extraction, SQL execution
2. **Direct queries** (`/query`): Accept POST requests with `{"sql": "...", "params": [...]}`  
3. **Proxy requests** (`/proxy/host.com/path`): Forward to `https://host.com/path` with CORS headers
4. **Static files** (`/*`): Serve from current directory, with `index.html` for root

### Database Integration

**SQLite**: `sqlite3_enable_load_extension(db, 1)` for extension support. Sets `MaxOpenConns(1)` and `MaxIdleConns(1)` for thread safety.

**PostgreSQL**: Standard `lib/pq` driver with automatic parameter placeholder conversion.

**Extension loading**: SQLite extensions loaded via `SELECT load_extension(path)` with proper file permissions and error handling.

### API Description Format

JSON structure defining API endpoints:
- `base_path`: Common prefix for all endpoints  
- `endpoints[].path`: URL pattern with `:param` placeholders
- `methods`: HTTP method → SQL query mapping
- `query` or `query_file`: Inline (SQL) query or external file reference
- `params[]`: Named parameter binding order

Path parameters (`{id}`) are extracted via regex and bound to SQL parameters in the order specified by the `params` array.
