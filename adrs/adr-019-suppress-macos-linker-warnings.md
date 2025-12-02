# ADR-019: Suppress macOS LC_DYSYMTAB Linker Warnings in CGo Builds

* **Status:** Accepted
* **Date:** 2025-12-02
* **Authors:** Claude, Mike Schinkel <mike@newclarity.net>

---

## Context

When building or testing the XMLUI test server on macOS with Go 1.25.x, the linker produces warnings like:

```
ld: warning: '/private/var/folders/.../000025.o' has malformed LC_DYSYMTAB,
expected 98 undefined symbols to start at index 884, found 95 undefined symbols
starting at index 884
```

These warnings appear specifically for packages using CGo, primarily `github.com/mattn/go-sqlite3` which is used for SQLite database support.

### Root Cause

The warnings stem from Apple's new `ld-prime` linker introduced in Xcode 15, which is more strict than the legacy `ld64` linker. The new linker validates Mach-O object file format more rigorously and issues warnings when it detects inconsistencies in the `LC_DYSYMTAB` (dynamic symbol table) load command.

When Go's CGo compiler generates object files, the symbol table structure doesn't perfectly match what Apple's strict linker expects. This is a known issue tracked in the Go project.

### Impact

- **Functional Impact:** None. These are cosmetic warnings only—binaries build correctly and function properly.
- **Developer Experience:** The warnings clutter test and build output, making it harder to spot real issues.
- **CI/CD:** Build logs become noisy, potentially masking important messages.

### Go Project Status

- Originally tracked in [golang/go#61229](https://github.com/golang/go/issues/61229)
- Partially fixed in Go 1.22
- Resurfaced in Go 1.25 ([golang/go#75274](https://github.com/golang/go/issues/75274))
- Complete fix scheduled for **Go 1.26** (CL 692996)
- Not backported to Go 1.25 because it's non-critical and required multiple iterations

---

## Decision

**Suppress linker warnings by setting `CGO_LDFLAGS="-Xlinker -w"` in the build and test scripts.**

### Implementation

1. **`scripts/shared.sh`**: Added `CGO_LDFLAGS` environment variable with default value `-Xlinker -w`
2. **`scripts/test.sh`**: Updated to pass `CGO_LDFLAGS` to `go test`
3. **`scripts/build.sh`**: Updated to pass `CGO_LDFLAGS` to `go build`

The flag `-Xlinker -w` passes the `-w` option directly to the linker, instructing it to suppress all warnings.

---

## Rationale

### Why Suppress Instead of Fix?

1. **Not Our Bug**: The issue is in Go's CGo compiler output interacting with Apple's linker—we can't fix it at the application level.

2. **Temporary Workaround**: The upstream fix will arrive in Go 1.26. This is a bridge solution.

3. **Zero Functional Impact**: The warnings are purely cosmetic. The binaries work correctly.

4. **Clean Builds**: Suppressing warnings improves developer experience and makes real issues more visible.

### Why `-Xlinker -w` Instead of Alternatives?

1. **Direct and Simple**: Passes the warning suppression flag directly to the linker.

2. **Well-Documented**: Standard approach recommended in Go and Xcode documentation for suppressing linker warnings.

3. **Override-able**: Using `${CGO_LDFLAGS:--Xlinker -w}` in `shared.sh` allows developers to override if needed.

4. **Minimal Scope**: Only affects CGo builds, doesn't change pure Go code compilation.

### Alternatives Considered

1. **Do Nothing**: Rejected because warnings clutter output and reduce signal-to-noise ratio.

2. **Upgrade to Go 1.26**: Not yet released (as of 2025-12-02).

3. **Selective Warning Suppression**: The linker doesn't support suppressing specific warning types—it's all or nothing.

4. **Switch to Pure Go SQLite**: Rejected due to performance implications and missing extension support.

---

## Consequences

### Positive

- **Clean Build Output**: Test and build commands no longer show linker warnings.
- **Better Developer Experience**: Easier to spot real issues in build logs.
- **CI/CD Improvement**: Build logs are cleaner and more actionable.
- **Future-Proof**: The fix can be easily removed once Go 1.26 is adopted.

### Negative

- **Suppresses All Linker Warnings**: We won't see legitimate linker warnings if they occur (though this is rare in Go projects).
- **macOS-Specific Workaround**: The configuration doesn't hurt other platforms but is only necessary on macOS.

### Migration Path

When upgrading to Go 1.26 or later:
1. Remove `CGO_LDFLAGS="-Xlinker -w"` from `scripts/shared.sh`
2. Test that builds and tests complete without warnings
3. Update this ADR status to "Superseded"

---

## References

### Go Project Issues

- [golang/go#61229](https://github.com/golang/go/issues/61229) - "cmd/link: issues with Apple's new linker in Xcode 15"
- [golang/go#75274](https://github.com/golang/go/issues/75274) - "cmd/link: ld warning: malformed LC_DYSYMTAB with -race" (Go 1.25)
- [golang/go#61558](https://github.com/golang/go/issues/61558) - Related LC_DYSYMTAB discussion

### Documentation

- [Stack Overflow: XCode 8 Linker (ld) suppress warnings](https://stackoverflow.com/questions/40331545/xcode-8-linker-ld-suppress-warnings)
- [Stack Overflow: How do you suppress GCC linker warnings?](https://stackoverflow.com/questions/3409448/how-do-you-suppress-gcc-linker-warnings)

### Changed Files

- `scripts/shared.sh` - Added `CGO_LDFLAGS` configuration
- `scripts/test.sh` - Updated to use `CGO_LDFLAGS`
- `scripts/build.sh` - Updated to use `CGO_LDFLAGS`

---

## Notes

This ADR documents a **temporary workaround** for a known upstream issue. Once Go 1.26 is released and adopted, this configuration should be removed and this ADR should be marked as "Superseded by Go 1.26 upgrade."

The decision to suppress warnings was made after confirming that:
1. The warnings are cosmetic and don't indicate real problems
2. The Go team acknowledges this as a known issue
3. A proper fix is coming in the next Go release
4. The suppression can be easily removed later
