# GoGoTrace

Go reverse call-graph analyzer that efficiently traces function dependencies across large codebases.

## Installation

```bash
go build -o gogotrace .
```

## Usage

```bash
gogotrace -func "<function signature>" [options]
```

### Options

- `-func string`: Function signature to trace (required)
- `-dir string`: Directory to analyze (default ".")
- `-json string`: Output results to JSON file
- `-html string`: Output results to HTML file
- `-no-test`: Exclude test functions from results
- `-list string`: List functions matching pattern
- `-debug`: Show debug information
- `-help`: Show help message


## Function Signature Format

The tool accepts complete function signatures:

- Simple functions: `func ProcessData()`
- With parameters: `func Calculate(a int, b string) error`
- Method receivers: `func (s *Server) Start(ctx context.Context)`
- Complex types: `func HandleRequest(ctx context.Context, req *http.Request)`

Note: Use exact signatures as they appear in code, including spacing.

