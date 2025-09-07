# GoGoTrace

Go reverse call-graph analyzer that efficiently traces function dependencies across large Go codebases. It scans your project, finds who calls a given function or method, and renders the reverse call tree to the console, JSON, or an interactive HTML page.

## Features

- Reverse call graph for a target function or method
- Fast parallel AST scanning across the repository
- Multiple outputs: console (default), JSON (`-json`), and interactive HTML (`-html`)
- Filter out test callers with `-no-test`
- Discover candidates with substring search using `-list`
- Helpful progress logs and `-debug` mode

## Installation

Build from source in this repository:

```bash
go build -o gogotrace .
```

Then run the binary with `./gogotrace` from your shell.

> Requires Go 1.21+ (see `go.mod`).

## Quick start

- Trace a simple function in the current directory:

```bash
./gogotrace -func "func main()"
```

- Trace a method in another project and exclude test callers:

```bash
./gogotrace -dir ~/code/myproject \
  -func "func (s *Server) Start(ctx context.Context)" \
  -no-test
```

- Export results:

```bash
# JSON
./gogotrace -func "func Process()" -json callgraph.json

# HTML (expand/collapse + search UI)
./gogotrace -func "func main()" -html callgraph.html
```

Console output example (indentation shows caller depth):

```
Call Graph:
===========
Service.Start in internal/service/service.go
    Server.Run in cmd/api/server.go
        main in cmd/api/main.go
```

## Usage

```bash
gogotrace -func "<function signature>" [options]
```

### Options

- `-func string`: Function signature to trace (required unless using `-list`)
- `-dir string`: Directory to analyze (default `.`)
- `-json string`: Write the call tree to a JSON file
- `-html string`: Write the call tree to an interactive HTML file
- `-no-test`: Exclude test functions from the results
- `-list string`: List functions whose name or signature contains the given substring
- `-debug`: Print extra call-graph details for the root and first-level callers
- `-help`: Show help and usage information

Examples:

```bash
./gogotrace -func "func (r *inboxMultiplexer) advanceSequencerMsg()"
./gogotrace -func "func WithAPIServer(api *server.Server) Opt" -no-test
./gogotrace -func "func Process()" -json output.json
./gogotrace -func "func main()" -html callgraph.html
./gogotrace -dir ~/myproject -func "func Init()" -no-test
```

### Function signature format and matching

Provide the target as a Go-style signature string. Return types are not part of matching (only name, optional receiver type, and parameter types):

- Simple functions: `func ProcessData()`
- With parameters: `func Calculate(a int, b string)`
- Method receivers: `func (s *Server) Start(ctx context.Context)`
- Complex types: `func Handle(ctx context.Context, req *http.Request)`

Matching details:

- Parameter names are ignored; only the types must match and in the same order.
- Receiver matching uses the receiver type (pointer vs non-pointer is normalized by type when comparing).
- Whitespace differences are ignored.
- Return types are ignored (signatures in this tool do not include returns).

Tip: If you are unsure of the exact signature, run `-list "Name"` to find candidates:

```bash
./gogotrace -dir ~/code/myproject -list "Start"
```

## Output formats

### Console (default)
Indented tree of callers for your target function, printed to stdout.

### HTML (`-html path`)
Writes an interactive page with:
- Expand/collapse controls
- Search box to highlight matching function names
- Per-node details: function/method name, package/file, usage counts, and a `TEST` badge for test callers

Open the generated file in your browser:

```bash
open callgraph.html
```

### JSON (`-json path`)
Machine-readable tree. Example shape:

```json
{
  "name": "Start",
  "receiver": "*Server",
  "package": "github.com/your/module/internal/service",
  "file": "service.go",
  "line": 42,
  "signature": "func (s *Server) Start(ctx context.Context)",
  "isTest": false,
  "children": [
    {
      "name": "Run",
      "receiver": "*Server",
      "package": "github.com/your/module/cmd/api",
      "file": "server.go",
      "line": 10,
      "signature": "func (s *Server) Run()",
      "usages": 1,
      "isTest": false,
      "children": [
        { "name": "main", "package": "github.com/your/module/cmd/api", "file": "main.go", "line": 5, "signature": "func main()" }
      ]
    }
  ]
}
```

## How it works (high level)

- Scans the target directory (recursively), excluding `vendor/`, `.git/`, `testdata/`, and `.work/`.
- Skips generated files ending in `.pb.go` and `_gen.go`.
- Phase 1 (parallel): extracts all function and method declarations.
- Phase 2 (parallel): builds a reverse call graph by scanning function bodies for call sites.
- Constructs a reverse call tree from the target up through its callers, recursively.

## Notes and limitations

- Best-effort static analysis based on the Go AST; it does not type-check or fully resolve packages.
- Method resolution uses naming heuristics for receiver identification; interface dispatch and some dynamic patterns may be missed.
- Results can include false positives/negatives in complex scenarios.
- Depth is bounded to avoid runaway graphs in large projects (current limit ~20 levels).

## Troubleshooting

- If no callers are found, verify the signature using `-list`, and ensure you are analyzing the correct `-dir`.
- Use `-no-test` to filter out noise from tests when exploring production call paths.
- Use `-debug` to print extra diagnostics for the root-level analysis.


## Demo

- HTML preview: docs/demo.html
- Open locally: `open docs/demo.html` (macOS) or serve the repo and browse to `/docs/demo.html`.
