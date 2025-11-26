# ADR-008: SQL Placeholder Syntax, Parsing, and Rewriting

## Status

Accepted (Amended 2025-11-26: See Amendment section below for syntax change)

## Context

The XMLUI test server allows developers to define APIs using JSON. Each API maps HTTP path/query/body parameters into SQL queries. These queries are executed against a database (initially SQLite, with planned support for DuckDB, PostgreSQL, MySQL, MariaDB, and maybe others).

A key requirement is a **safe, consistent way to represent query parameters in SQL templates**:

* Parameters come from **path**, **query string**, or **JSON body**.
* The server must bind parameters safely (avoiding SQL injection) and rewrite placeholders to the **native syntax of the target backend**.
* Placeholders must be **named** (not positional) because HTTP parameters do not have a guaranteed positional order.
* Developers using this system may be relatively unskilled, so the syntax must be unambiguous, easy to learn, and resistant to accidental collisions.

Several styles exist in the ecosystem:

* `?` positional parameters (JDBC, MySQL, SQLite)
* `$1, $2, …` positional parameters (PostgreSQL)
* `:name` named parameters (PDO, Oracle, ActiveRecord)
* `@name` named parameters (SQL Server, ADO.NET)

Challenges with adopting `:name` directly:

* `:` is common in string literals (e.g. times like `08:30`, JSON keys like `"foo:bar"`).
* Avoiding false positives would require SQL parsing or complex heuristics.

To minimize complexity while ensuring correctness, a brace-delimited form (`:name`) was considered and found preferable.

Additionally, optional parameters and default values are already handled **upstream** in the URL parameter parser (using syntax like `{name:type:constraints}`, `{name?}`, and `{name?default}`). Duplicating this logic in the SQL layer would add confusion and unnecessary complexity.

## Decision

* **Canonical placeholder syntax**:
  Use **brace-delimited named placeholders**:

  ```
  :name
  :payload.user.id
  :items[0].sku
  :body.event
  ```

  Placeholders may include **dot-separated names** and **array indices** (using bracket notation) to traverse JSON request bodies, enabling expressions like `:user.id` or `:payload.items[0].sku`.

* **Scope of SQL layer**:

  * SQL placeholder processing is responsible **only** for:

    * Detecting `{…}` placeholders outside of string literals, identifiers, and comments.
    * Rewriting placeholders into backend-native parameter syntax.
    * Producing an ordered parameter list for binding.
  * The SQL layer does **not** implement optionality, defaults, or constraints.
    These are resolved **upstream** in the URL/JSON parameter resolution step.

* **Parser & rewrite API**:
  The SQL parser is implemented as `ParseSQL(sql string, args ParseSQLArgs) (PreparedSQL, error)`.

  * `PreparedSQL` contains the rewritten SQL and the ordered list of parameters.
  * `ParseSQLArgs` requires a `FormatParamFunc func(idx int) string` which generates the backend-specific placeholder (`$1`, `?`, `@p1`, etc.).
  * This isolates backend differences in one function, keeping the parser database-agnostic and extensible (e.g., adding DuckDB support later).

* **Rewriting rules**:

  * PostgreSQL → `$1`, `$2`, …
  * MySQL → `?`
  * SQLite → `?`
  * SQL Server → `@p1`, `@p2`, …

* **Duplicate placeholders**:
  The same `:name` may appear multiple times in SQL. It will be bound once and substituted consistently.

* **Invalid placeholders**:
  Placeholders not resolved by upstream parameter handling result in an error at execution time.

* **Array expansion**:
  Reserved for future handling. Expansion (`IN (?)`) will be addressed upstream in the parameter resolver before SQL rewriting.

* **Superset scanning**:
  The parser always recognizes and skips all major SQL literal/identifier/comment forms (e.g. `'…'`, `"…"`, `` `…` ``, `[ … ]`, `-- …`, `# …`, `/* … */`, Postgres dollar-quotes `$tag$…$tag$`, Oracle `q'…'` quoting).
  This “safe superset” approach avoids false matches inside these regions and works without conflict across SQLite, PostgreSQL, MySQL/MariaDB, SQL Server, DuckDB, Oracle, and DB2.

## Consequences

* **Pros**:

  * Clear, unambiguous syntax unlikely to appear accidentally inside SQL literals.
  * Keeps SQL processing simple (tokenizer, not full parser).
  * Consistent with the project’s existing brace-based variable syntax for URL templates.
  * Easy to extend later with annotations (`{name:uuid}`) if needed.
  * Decouples responsibilities: upstream handles typing, constraints, and defaults; SQL layer focuses solely on rewriting.

* **Cons**:

  * Different from common `:name` style used in many frameworks, so developers may need to adapt.
  * Requires a custom tokenizer to skip string/comment regions (though far simpler than a full SQL parser).
  * Array expansion support must be designed carefully in the upstream resolver.

* **Neutral**:

  * Optional/default handling at the SQL layer was consciously excluded to avoid duplication, even though some systems (e.g., Rails) allow it in queries. This choice favors clarity over familiarity.

## Alternatives Considered

* **Use `:name` directly**
  Rejected due to parsing ambiguity inside strings and potential collisions with casts (`::`) and assignments (`:=`).
* **Use positional placeholders (`?`, `$1`, etc.) only**
  Rejected because HTTP parameters have no stable positional ordering; named placeholders are required.
* **Include optional/default support in SQL placeholders**
  Rejected because the URL parameter system already handles these; duplication would be confusing and error-prone.

## Still To Be Decided

* Define a syntax for **array expansion** in `IN` lists (e.g., `{ids[]}`, `:ids...` or other?).
* Extend to support more backends as needed.

## Future Work

* Add optional support for **inline type annotations** (e.g., `{id:uuid}`) to control SQL casts (but probably not).

## Examples

### Example 1: Simple equality

**API SQL template:**

```sql
SELECT * FROM users WHERE id = :id;
```

**PostgreSQL (rewritten):**

```sql
SELECT * FROM users WHERE id = $1;
```

Args: `[42]`

**MySQL / SQLite (rewritten):**

```sql
SELECT * FROM users WHERE id = ?;
```

Args: `[42]`

**SQL Server (rewritten):**

```sql
SELECT * FROM users WHERE id = @p1;
```

Args: `[42]`

---

### Example 2: Multiple parameters with reuse

**API SQL template:**

```sql
SELECT * FROM orders
WHERE account_id = :accountId
  AND created_at >= :since
  AND updated_at >= :since;
```

**PostgreSQL:**

```sql
SELECT * FROM orders
WHERE account_id = $1
  AND created_at >= $2
  AND updated_at >= $2;
```

Args: `[acct123, "2024-01-01T00:00:00Z"]`

**MySQL / SQLite:**

```sql
SELECT * FROM orders
WHERE account_id = ?
  AND created_at >= ?
  AND updated_at >= ?;
```

Args: `[acct123, "2024-01-01T00:00:00Z"]`

**SQL Server:**

```sql
SELECT * FROM orders
WHERE account_id = @p1
  AND created_at >= @p2
  AND updated_at >= @p2;
```

Args: `[acct123, "2024-01-01T00:00:00Z"]`

---

### Example 3: Dotted path

**API SQL template:**

```sql
INSERT INTO events (user_id, payload)
VALUES (:user.id, :body.event);
```

**PostgreSQL:**

```sql
INSERT INTO events (user_id, payload)
VALUES ($1, $2);
```

Args: `[resolvedUserID, {"type":"click","target":"btn"}]`

**MySQL / SQLite:**

```sql
INSERT INTO events (user_id, payload)
VALUES (?, ?);
```

**SQL Server:**

```sql
INSERT INTO events (user_id, payload)
VALUES (@p1, @p2);
```

---

## Amendment (2025-11-26): Migration to Colon-Prefixed Syntax

### Context

After implementation and usage, the brace-delimited syntax `:name` introduced IDE compatibility issues:

* Most SQL IDEs and editors flag `:name` as syntax errors since it's not standard SQL
* This creates visual noise and reduces developer productivity
* The `:name` syntax is widely recognized by SQL tooling as a valid parameter placeholder

The original concerns about `:name` (collisions with time literals and casts) were addressable through proper parsing.

### Decision

**Migrated from `:name` to `:name` syntax** (colon-prefixed placeholders).

**Rationale:**
* Standard SQL tooling recognizes `:name` as parameter syntax
* IDE syntax highlighting works correctly
* Better alignment with SQL ecosystem conventions
* Parser successfully handles edge cases:
  - PostgreSQL `::` type casts (explicitly detected and skipped)
  - Time literals in strings (`'12:30:00'` already handled by string skip logic)
  - Colons in comments and identifiers (already handled by superset scanning)

**Updated canonical syntax:**
```
:name
:payload.user.id
:items[0].sku
:body.event
```

All other design decisions remain unchanged:
* Dot-separated paths for JSON traversal
* Duplicate placeholder reuse
* Backend-agnostic parser with `FormatParamFunc`
* Superset scanning to avoid false matches

### Implementation

Parser changes:
* State machine now detects `:` followed by valid identifier characters
* Explicit handling for PostgreSQL `::` operator (skip both colons)
* Simpler parsing logic (no closing delimiter needed)
* Natural word boundaries for parameter extraction

### Updated Examples

**Example 1: Simple equality**
```sql
SELECT * FROM users WHERE id = :id;
```

**Example 2: Multiple parameters with reuse**
```sql
SELECT * FROM orders
WHERE account_id = :accountId
  AND created_at >= :since
  AND updated_at >= :since;
```

**Example 3: Dotted path**
```sql
INSERT INTO events (user_id, payload)
VALUES (:user.id, :body.event);
```

**Example 4: PostgreSQL type casts (no conflict)**
```sql
SELECT name::text, created_at::date
FROM users
WHERE id = :userId;
```

### Migration Impact

* All SQL query templates updated from `:param` to `:param`
* All configuration files migrated
* All tests updated and passing
* Zero breaking changes to parameter resolution logic
* Improved developer experience with IDE compatibility
