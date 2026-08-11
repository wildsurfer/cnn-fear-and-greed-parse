# cnn-fear-and-greed-parse

[![Go Reference](https://pkg.go.dev/badge/github.com/wildsurfer/cnn-fear-and-greed-parse/v2.svg)](https://pkg.go.dev/github.com/wildsurfer/cnn-fear-and-greed-parse/v2) ![CI](https://github.com/wildsurfer/cnn-fear-and-greed-parse/actions/workflows/go.yml/badge.svg) [![Coverage Status](https://coveralls.io/repos/github/wildsurfer/cnn-fear-and-greed-parse/badge.svg?branch=main)](https://coveralls.io/github/wildsurfer/cnn-fear-and-greed-parse?branch=main) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Go package for CNN's [Fear & Greed Index](https://www.cnn.com/markets/fear-and-greed), a 0–100 gauge of US stock market sentiment. It returns the current score and rating, the values for the previous close, week, month and year, about a year of daily history, and the seven component indicators (market momentum, stock price strength, stock price breadth, put/call options, market volatility, junk bond demand, safe haven demand), each with its own score, rating and history of raw values. The module also ships a CLI and an MCP server, and has no dependencies outside the Go standard library.

## Install

```
go get github.com/wildsurfer/cnn-fear-and-greed-parse/v2
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
)

func main() {
	result, err := cnnfag.Get(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Now: %.0f (%s)\n", result.Score, result.Rating)
	fmt.Printf("One year ago: %.0f\n", result.OneYearAgo)
	fmt.Printf("History: %d daily points since %s\n",
		len(result.History), result.History[0].Date.Format("2006-01-02"))
	fmt.Printf("VIX indicator: %.0f (%s)\n",
		result.MarketVolatility.Score, result.MarketVolatility.Rating)
}
```

Output:

```
Now: 64 (greed)
One year ago: 58
History: 250 daily points since 2025-08-11
VIX indicator: 50 (neutral)
```

An `Indicator`'s `History` holds the raw underlying series (the S&P 500 level for momentum, the VIX level for volatility, ratios and spreads for the rest), and its `Score` is CNN's 0–100 normalization. To use your own `http.Client` (timeout, proxy), replace `cnnfag.HTTPClient`.

## CLI

For cron jobs and shell pipelines, without writing Go:

```
go install github.com/wildsurfer/cnn-fear-and-greed-parse/v2/cmd/cnnfag@latest
```

```
$ cnnfag
64 (greed) as of 2026-08-11T00:00:00Z
previous close 64 · week ago 60 · month ago 47 · year ago 58

$ cnnfag -json | jq .score
64.3714285714286
```

`-json` prints the full result, including the daily history. `-timeout` changes the request timeout (default 15s).

## MCP server

`cnnfag mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io) server over stdio, so AI assistants can query the index. It exposes one tool, `get_fear_and_greed`, with an optional `include_history` argument. Configuration for MCP clients:

```json
{
  "mcpServers": {
    "cnnfag": {
      "command": "cnnfag",
      "args": ["mcp"]
    }
  }
}
```

The client must be able to find the binary: use the full path (usually `~/go/bin/cnnfag`) if your MCP client does not inherit your shell's `PATH`. Like the rest of the module, the server is built on the standard library only.

The server is also published to the [MCP Registry](https://registry.modelcontextprotocol.io) as `io.github.wildsurfer/cnnfag`, with a container image at `ghcr.io/wildsurfer/cnnfag` for clients that prefer Docker over a local binary.

## How it works

CNN does not offer a documented public API. This package requests the JSON endpoint that the Fear & Greed page itself uses:

```
https://production.dataviz.cnn.io/index/fearandgreed/graphdata
```

The endpoint rejects requests that do not look like they come from a browser, so the package sends browser-like `User-Agent` and `Referer` headers. This is the same data source used by the known wrappers in other languages.

A scheduled CI job runs the test suite against the real endpoint once a week, so a change on CNN's side is detected within days.

## Migrating from v1

v2 is a full rewrite: CNN removed the HTML page that v1 parsed, so v1 stopped working and its data model no longer matches what CNN publishes. Update the import path first:

```go
import cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
```

Then adjust the calls:

| v1 | v2 |
|----|----|
| `cnnfag.Parse()` | `cnnfag.Get(ctx)`, takes a `context.Context` |
| `Result.Now.Value` (`int`) | `Result.Score` (`float64`) |
| `Result.Now.Text` | `Result.Rating` |
| `Result.PreviousClose.Value` (`int`) | `Result.PreviousClose` (`float64`) |
| `Result.OneWeekAgo.Value`, `.OneMonthAgo.Value`, `.OneYearAgo.Value` | same field names, now plain `float64` |
| `Result.PreviousClose.Text` and other past-period labels | removed; CNN's API has no labels for past periods, but every `History` point carries a rating |
| `Result.LastUpdateDate` | `Result.Timestamp`, now an exact time from CNN instead of a parsed guess |
| `Result.ImageURL`, `Result.GetImageBytes()` | removed; the needle image no longer exists |
| `ErrHTTPNon200` | `ErrUnexpectedStatus`, check with `errors.Is` |
| `ErrEmptyField` | `ErrEmptyResult` |
| `ErrImgLoadNon200`, `ErrReadingBytes` | removed with the image API |
| — | `Result.History` is new: about a year of daily scores |

Scores changed from rounded integers to the exact floats CNN serves, so `44` in v1 corresponds to something like `43.71` in v2. Round with `%.0f` or `math.Round` if you need the old look.

## Project history

v1 (2021) parsed the HTML of `money.cnn.com/data/fear-and-greed` with goquery and could also download the index needle image. CNN removed that page, which broke parsing, and the image no longer exists. v2 (2026) is a rewrite on the JSON endpoint with a smaller API, historical data and zero dependencies. The last v1 release is tagged [`v1.2.0`](https://github.com/wildsurfer/cnn-fear-and-greed-parse/tree/v1.2.0).

## Data disclaimer

This package is not affiliated with or endorsed by CNN. The Fear & Greed Index and its values belong to CNN (Warner Bros. Discovery). The MIT license covers only the code in this repository and gives you no rights to CNN's data. CNN's Terms of Use permit personal use of site content and restrict commercial exploitation. If you use this data in a product, compliance is your responsibility, and the endpoint can change or disappear at any time.

## License

[MIT](LICENSE)
