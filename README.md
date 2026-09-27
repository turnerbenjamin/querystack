# QueryStack

 A Go library to parse, plan and execute resource-style queries, produce
 JSON results and handle pagination tokens. Designed for pluggable data
 repositories and SQL writers (currently supports Azure SQL).

## Features

- Parse resource query strings into an AST and plan execution
- Generate SQL for planned queries (Azure SQL writer included)
- Execute queries against a pluggable `Repository` that returns JSON
- Materialise JSON into typed table models
- Cursor-style pagination tokens with signing
- Configurable limits and defaults via `QueryConfig`

## Install

 ```bash
 go get github.com/turnerbenjamin/querystack
 ```
