# cnn-fear-and-greed-parse

[![Go Reference](https://pkg.go.dev/badge/github.com/wildsurfer/cnn-fear-and-greed-parse/v2.svg)](https://pkg.go.dev/github.com/wildsurfer/cnn-fear-and-greed-parse/v2) ![CI](https://github.com/wildsurfer/cnn-fear-and-greed-parse/actions/workflows/go.yml/badge.svg) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Go package for CNN's [Fear & Greed Index](https://www.cnn.com/markets/fear-and-greed), a 0–100 gauge of US stock market sentiment. It returns the current score and rating, the values for the previous close, week, month and year, and about a year of daily history. The package has no dependencies outside the Go standard library.

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
}
```

Output:

```
Now: 64 (greed)
One year ago: 58
History: 250 daily points since 2025-08-11
```

## How it works

CNN does not offer a documented public API. This package requests the JSON endpoint that the Fear & Greed page itself uses:

```
https://production.dataviz.cnn.io/index/fearandgreed/graphdata
```

The endpoint rejects requests that do not look like they come from a browser, so the package sends browser-like `User-Agent` and `Referer` headers. This is the same data source used by the known wrappers in other languages.

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
| `ErrHTTPNon200`, `ErrEmptyField`, `ErrImgLoadNon200`, `ErrReadingBytes` | `ErrUnexpectedStatus`, check with `errors.Is` |
| — | `Result.History` is new: about a year of daily scores |

Scores changed from rounded integers to the exact floats CNN serves, so `44` in v1 corresponds to something like `43.71` in v2. Round with `%.0f` or `math.Round` if you need the old look.

## Project history

v1 (2021) parsed the HTML of `money.cnn.com/data/fear-and-greed` with goquery and could also download the index needle image. CNN removed that page, which broke parsing, and the image no longer exists. v2 (2026) is a rewrite on the JSON endpoint with a smaller API, historical data and zero dependencies. The last v1 release is tagged [`v1.2.0`](https://github.com/wildsurfer/cnn-fear-and-greed-parse/tree/v1.2.0).

## License

[MIT](LICENSE)
