# Lightstep UQL to PromQL Translator

A Go library, HTTP server and UI for translating Lightstep UQL (Unified Query Language) queries to PromQL (Prometheus Query Language).

## Context

This tool was developed to automate the migration of thousands of telemetry queries and alerts from **Lightstep UQL** to **PromQL**-compatible backends. Since Lightstep is being sunset, we are sharing this to help the engineering community transition their observability stacks with minimal manual effort.

## Features

- **HTTP Server & Web UI** - Interactive web interface and REST API for query translation
- **UQL to PromQL Translation SDK** - Programmatic query translation
- **UQL Optimizer** - Query optimization before translation
- **Streams Parser** - Parse stream filter queries
- **UQL Lexer** - Tokenization of UQL queries
- **UQL Parser** - AST generation from UQL queries

## Installation

```bash
go get github.com/zalando/lightstep-uql-to-promql-translator
```

## Usage

The tool is designed to be dual-purpose:

1. **As a Service:** A ready-to-go HTTP server with a Web UI and REST API. Best for interactive use or shared migration portals.
2. **As an SDK:** A Go library for programmatic translation. Best for embedding conversion logic into your own internal tools or CI pipelines.

### 1. HTTP Server & Web UI

The easiest way to use the translator is through the built-in HTTP server, which provides both a web UI for interactive translation and a REST API for programmatic access.

#### Running the Server

1. Configure your metric types and special metrics in `cmd/main.go`:

```go
package main

import (
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/promql"
    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/server"
)

func TranslateUQLToPromQL(query string) (string, *model.Error) {
    metricTypes := map[string]promql.MetricType{
        "http_requests_total": promql.METRIC_TYPE_SUM,
        "cpu_usage":           promql.METRIC_TYPE_GAUGE,
        "request_duration":    promql.METRIC_TYPE_HISTOGRAM,
    }

    metricConfig := promql.SpecialMetricConfig{
        SpansCount:           "spans.count",
        SpansLatency:         "spans.latency",
        SpansCountUnadjusted: "spans.count_unadjusted",
        LogsCount:            "logs.count",
    }

    return promql.Translate(query, metricTypes, metricConfig)
}

func main() {
    srv := server.New(":8080", TranslateUQLToPromQL)
    log.Fatal(srv.Start())
}
```

2. Start the server:

```bash
go run cmd/main.go
```

The server will start on port 8080 (or your configured port).

#### Web UI

The server provides an interactive web interface for translating UQL queries to PromQL. Simply open your browser and navigate to the server URL:

```bash
# Open the web UI
open http://localhost:8080
```

The Web UI allows you to:

- Enter UQL queries in a text editor
- See real-time PromQL translation results
- View detailed error messages with source position highlighting
- Experiment with different query patterns

#### REST API

**Endpoint:** `POST /api/translate`

**Request:**

```json
{
    "query": "metric http_requests_total | rate 5m | group_by [method], sum"
}
```

**Response (Success):**

```json
{
    "promql": "sum by (method) (rate(http_requests_total[5m]))",
    "error": null
}
```

**Response (Error):**

```json
{
    "promql": "spans% count | ...",
    "error": {
        "status": "unexpected token",
        "source_index": 5,
        "source_length": 1
    }
}
```

#### Using curl

```bash
curl -X POST http://localhost:8080/api/translate \
     -d '{"query": "metric http_requests_total | rate 5m"}'
```

### 2. UQL to PromQL Translation SDK

For programmatic translation in your Go applications, use the translation SDK directly.

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/promql"
)

func main() {
    query := "metric my_histogram | delta"

    metricTypes := map[string]promql.MetricType{
        "my_sum_metric":       promql.METRIC_TYPE_SUM,
        "my_gauge_metric":     promql.METRIC_TYPE_GAUGE,
        "my_histogram_metric": promql.METRIC_TYPE_HISTOGRAM,
    }

    metricConfig := promql.SpecialMetricConfig{
        SpansCount:           "my.spans.count",
        SpansLatency:         "my.spans.latency",
        SpansCountUnadjusted: "my.spans.count_unadjusted",
        LogsCount:            "my.logs.count",
    }

    promqlQuery, err := promql.Translate(query, metricTypes, metricConfig)
    if err != nil {
        log.Fatalf("Translation error: %s", err.Status)
    }

    fmt.Println("PromQL:", promqlQuery)
}
```

### 3. UQL Optimizer

The optimizer applies various transformations to UQL queries to improve translation efficiency and output quality. It can merge filter stages, convert expressions to disjunctive normal form, and apply other optimizations.

#### Basic Optimization

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/optimizer"
)

func main() {
    query := "metric my_metric | filter status == 200 | filter method == GET"

    // Optimize with default configuration
    optimizedAst, err := optimizer.Optimize(query)
    if err != nil {
        log.Fatalf("Optimization error: %s at position %d", err.Status, err.SourceIndex)
    }

    fmt.Println("Optimized AST:")
    fmt.Println(optimizedAst.ToXml())
}
```

#### Custom Optimization Configuration

You can customize which optimizations are applied:

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/optimizer"
    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/parser"
)

func main() {
    query := "metric my_metric | filter status == 200 || status == 201 || status == 204"

    queryAst, err := parser.Parse(query)
    if err != nil {
        log.Fatal(err)
    }

    // Create custom optimization config
    config := optimizer.OptimizerConfig{
        Filter: optimizer.OptimizerConfigFilter{
            MergeStagesIntoSingleStage:                true,
            ConvertSingleAttributeConjunctionToRegexp: true,
            PushLogicalNegationsDownExpressionTree:    true,
            ConvertExpressionToDisjunctiveNormalForm:  true,
            ConvertContainsOperationToRegexp:          true,
            ConvertPhraseMatchOperationToRegexp:       true,
        },
        PointFilter: optimizer.OptimizerConfigPointFilter{
            ConvertSingleAttributeConjunctionToRegexp: true,
            PushLogicalNegationsDownExpressionTree:    true,
            ConvertExpressionToDisjunctiveNormalForm:  true,
            ConvertContainsOperationToRegexp:          true,
        },
        CustomOptimizations: nil,
    }

    optimizedAst, err := optimizer.OptimizeQuery(queryAst, config)
    if err != nil {
        log.Fatalf("Optimization error: %s", err.Status)
    }

    fmt.Println("Optimized query with custom config")
    fmt.Println(optimizedAst.ToXml())
}
```

#### Available Optimizations

**Filter Optimizations:**

- `MergeStagesIntoSingleStage` - Combines multiple filter stages into one
- `ConvertSingleAttributeConjunctionToRegexp` - Converts OR operations on the same attribute to regex (e.g., `status == 200 || status == 201` → `status =~ "^(200|201)$"`)
- `PushLogicalNegationsDownExpressionTree` - Moves NOT operations closer to operands using De Morgan's laws
- `ConvertExpressionToDisjunctiveNormalForm` - Converts expressions to DNF for better optimization
- `ConvertContainsOperationToRegexp` - Converts contains operations to regex patterns
- `ConvertPhraseMatchOperationToRegexp` - Converts phrase match operations to regex patterns

**Point Filter Optimizations:**

- Same optimizations as filters (except `MergeStagesIntoSingleStage`)

### 4. Streams Parser

The streams parser processes stream filter queries, extracting filter conditions into a structured format.

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/streams"
)

func main() {
    streamQuery := `service IN ("api", "ui") AND status IN ("200", "201")`

    // Parse the stream query
    filters, err := streams.Parse(streamQuery)
    if err != nil {
        log.Fatalf("Parse error: %s at position %d", err.Status, err.SourceIndex)
    }

    fmt.Println("Parsed filters:")
    for _, filter := range filters {
        fmt.Printf("  Key: %s, Operator: %s, Values: %v\n",
            filter.Key, filter.Operator, filter.Values)
    }
}
```

### 5. UQL Lexer

The lexer tokenizes UQL query strings into a sequence of tokens. Use this if you need low-level access to the token stream.

#### Basic Tokenization

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/lexer"
)

func main() {
    query := "metric my_metric | rate 1m | group_by [service], sum"

    // Tokenize without comments
    tokens, err := lexer.Tokenize(query)
    if err != nil {
        log.Fatalf("Lexer error: %s at position %d", err.Status, err.SourceIndex)
    }

    fmt.Println("Tokens:")
    for _, token := range tokens {
        fmt.Printf("  Type: %-20s Value: %s\n", token.Type, token.Value)
    }
}
```

#### Tokenizing with Comments

If you need to preserve comments in the token stream:

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/lexer"
)

func main() {
    query := `# This is a comment
    metric my_metric | rate`

    // Tokenize including comments
    tokens, err := lexer.TokenizeWithComments(query)
    if err != nil {
        log.Fatalf("Lexer error: %s", err.Status)
    }

    for _, token := range tokens {
        fmt.Printf("%s: %s\n", token.Type, token.Value)
    }
}
```

#### Using the Lexer Directly

For streaming token processing:

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/lexer"
    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

func main() {
    query := "metric my_metric | rate"

    lex := lexer.New(query)

    for {
        token, err := lex.FetchNextToken()
        if err != nil {
            log.Fatalf("Lexer error: %s", err.Status)
        }

        if token.Type == model.TypeEOF {
            break
        }

        fmt.Printf("Token: %s = %s\n", token.Type, token.Value)
    }
}
```

### 6. UQL Parser

The parser converts UQL queries into an Abstract Syntax Tree (AST). The AST structures are defined in `pkg/model/ast`.

#### Basic Parsing

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/parser"
)

func main() {
    query := "metric http_requests_total | rate 5m | group_by [method], sum"

    // Parse the query into an AST
    ast, err := parser.Parse(query)
    if err != nil {
        log.Fatalf("Parser error: %s at position %d", err.Status, err.SourceIndex)
    }

    // Access AST properties
    fmt.Println("Query Type:", ast.Type)
    fmt.Println("Pipeline Stages:", len(ast.Pipeline))

    // Export to XML for debugging
    fmt.Println("\nAST as XML:")
    fmt.Println(ast.ToXml())
}
```

#### Working with the AST

The AST provides structured access to query components:

```go
package main

import (
    "fmt"
    "log"

    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/parser"
    "github.com/zalando/lightstep-uql-to-promql-translator/pkg/model/ast"
)

func main() {
    query := "metric my_metric | rate | group_by [cluster], sum"

    queryAst, err := parser.Parse(query)
    if err != nil {
        log.Fatal(err)
    }

    // Traverse the pipeline stages
    for i, stage := range queryAst.Pipeline {
        fmt.Printf("Stage %d: %T\n", i, stage)

        // Type assertion to access stage-specific properties
        switch s := stage.(type) {
        case ast.FetchStage:
            fmt.Println("  - This is a fetch stage")
        case ast.AlignerStage:
            fmt.Println("  - This is an aligner stage")
        case ast.ModifierStage:
            fmt.Println("  - This is a modifier stage")
        default:
            fmt.Printf("  - Unknown stage type: %T\n", s)
        }
    }
}
```

## Error Handling

All parsing and translation functions return a `*model.Error` type that includes:

- `Status` - Error message
- `SourceIndex` - Position in the query where the error occurred
- `SourceLength` - Length of the problematic token

```go
promqlQuery, err := promql.Translate(query, metricConfig)
if err != nil {
    fmt.Printf("Error: %s\n", err.Status)
    fmt.Printf("Location: characters %d-%d\n", err.SourceIndex, err.SourceIndex+err.SourceLength)
}
```

## Project Status

This project is production-ready and currently used internally at Zalando to power our telemetry migration.
While fully functional, we are working toward a **v1.1 release** to reach our "Golden Standard" for open source.

**Planned for v1.1:**

- **CI/CD:** GitHub Actions for automated testing and linting.
- **Extended Testing:** Expanded unit test coverage and Lexer/Parser Fuzz testing.

## License

See LICENSE file for details.
