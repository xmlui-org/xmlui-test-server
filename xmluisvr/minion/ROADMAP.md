# Minion Package Roadmap

## Potential Features (Unlikely to Implement)

These features are documented for reference in case future needs arise. They are not currently prioritized for implementation.

### Multi-Format Rendering Helpers

Add convenience methods on the `Demos` type for rendering table output in different formats:

```go
// RenderText renders demos as plain text table to io.Writer
func (ds Demos) RenderText(w io.Writer, args DemoTableWriterArgs) error

// RenderMarkdown renders demos as Markdown table to io.Writer
func (ds Demos) RenderMarkdown(w io.Writer, args DemoTableWriterArgs) error

// RenderHTML renders demos as HTML table to io.Writer
func (ds Demos) RenderHTML(w io.Writer, args DemoTableWriterArgs) error
```

**Why unlikely:**
- Core functionality already supports flexible table rendering through `TableWriter()`
- Callers can render to any format using go-pretty's built-in methods on the returned `table.Writer`
- These would only provide convenience shortcuts, not new functionality
- No identified use case for multi-format rendering in current tooling

**Implementation approach (if needed):**
1. Call `ds.TableWriter(args)` to get configured table
2. Use go-pretty's built-in methods: `RenderMarkdown()`, `RenderHTML()`, `Render()`
3. Write result to provided `io.Writer`
4. Return any errors from the Writer

**When to reconsider:**
- If documentation generation needs table output in multiple formats
- If external tools request standardized rendering methods
- If CLI commands need to generate reports in different formats
