# GoGoTrace

GoGoTrace is a Go reverse call‑graph analyzer. It scans a project, discovers who calls a given function or method, and renders a reverse call tree to the console, to JSON, or to an interactive HTML page. The analysis runs in parallel and is designed to handle large codebases efficiently.

## Motivation


## Motivation

I built GoGoTrace because I could not find a Go tool that lets me start from an exact function signature and deterministically trace all of its callers, recursively, as a proper reverse call chain. Tools like go-callvis are useful for high‑level visualization, but they primarily produce an HTML call graph and typically require you to point at whole packages; they do not let you backtrace from a single function the way a debugger backtrace does. GoGoTrace fills that gap.

It also fits LLM‑driven workflows. You can ask an LLM to inspect a particular function and then, using GoGoTrace’s reverse call tree as context, reason about how upstream callers may or may not mitigate an issue. For example, you might prompt: “I believe there’s a bug in this function; search in every calling function whether there is a mitigation that would stop this bug.” Because LLMs are not reliable at exhaustively and deterministically enumerating callers, supplying the concrete call tree improves both accuracy and speed.

If you have other use cases in mind, I’d love to hear them.

## Installation

Build the CLI from source in this repository, then run the binary from your shell.

```bash
go build -o gogotrace .
```

Go 1.21 or newer is required (see `go.mod`).

## Quick start

Run the tool against the current directory and trace a simple function:

```bash
./gogotrace -func "func main()"
```

Point the tool at another project, trace a method by its signature, and exclude test callers:

```bash
./gogotrace -dir ~/code/myproject -func "func (s *Server) Start(ctx context.Context)" -no-test
```

Export results either as JSON or as an interactive HTML page:

```bash
./gogotrace -func "func Process()" -json callgraph.json
./gogotrace -func "func main()" -html callgraph.html
```

Console output is an indented tree where depth reflects distance from the target function:

```
Call Graph:
===========
Service.Start in internal/service/service.go
    Server.Run in cmd/api/server.go
        main in cmd/api/main.go
```

## Usage

The general form is `gogotrace -func "<function signature>" [options]`.

The most important flag is `-func`, which specifies the function or method signature to trace. The `-dir` flag sets the directory to analyze and defaults to the current directory. The tool can write JSON to a file via `-json <path>` and an interactive HTML page via `-html <path>`. Test callers can be removed from the output with `-no-test`. If you are unsure of the exact signature, use `-list <substring>` to print functions whose name or signature contains that substring. Extra diagnostics can be enabled with `-debug`. Pass `-help` to print the built‑in usage summary.

Here are several concrete invocations:

```bash
./gogotrace -func "func (r *inboxMultiplexer) advanceSequencerMsg()"
./gogotrace -func "func WithAPIServer(api *server.Server) Opt" -no-test
./gogotrace -func "func Process()" -json output.json
./gogotrace -func "func main()" -html callgraph.html
./gogotrace -dir ~/myproject -func "func Init()" -no-test

## Output formats

The console view (the default) prints a readable tree to standard output. The HTML view (`-html <path>`) writes an interactive page that supports expanding and collapsing nodes and a client‑side search box that highlights matching function names. The JSON view (`-json <path>`) writes a machine‑readable tree. A representative JSON fragment looks like the following:

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

## How it works

The analyzer walks the target directory recursively while skipping `vendor/`, `.git/`, `testdata/`, and `.work/`. Generated files ending in `.pb.go` and `_gen.go` are ignored. In the first phase, the tool extracts function and method declarations in parallel. In the second phase, it scans function bodies in parallel to assemble a reverse call graph. From the selected target signature, it then builds a reverse call tree by repeatedly finding callers, with cycle protection and a reasonable depth limit to keep results focused.

## Notes and limitations

This is a best‑effort static analysis based on the Go AST and does not perform full type checking or package resolution. Method resolution uses receiver‑name heuristics, which means dynamic dispatch through interfaces and some complex patterns may be missed. In very large or highly dynamic codebases the results can contain false positives or false negatives, but they are typically useful for navigation and impact analysis.

## Troubleshooting

If no callers are reported, confirm the exact signature using `-list` and double‑check the `-dir` value. When exploring production‑only paths, add `-no-test` to remove test callers. If you need extra detail while iterating, run with `-debug` to see information about the root and its immediate callers.

