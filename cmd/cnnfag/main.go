// Command cnnfag prints CNN's Fear & Greed index as text or JSON, and can run
// a Model Context Protocol server exposing the index as a tool ("cnnfag mcp").
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
)

func main() {
	jsonOut := flag.Bool("json", false, "print the full result, including history, as JSON")
	timeout := flag.Duration("timeout", 15*time.Second, "request timeout")
	flag.Parse()

	switch flag.Arg(0) {
	case "":
	case "mcp":
		if err := serveMCP(os.Stdin, os.Stdout, cnnfag.Get); err != nil {
			fmt.Fprintln(os.Stderr, "cnnfag mcp:", err)
			os.Exit(1)
		}
		return
	default:
		fmt.Fprintf(os.Stderr, "cnnfag: unknown command %q, the only command is \"mcp\"\n", flag.Arg(0))
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	res, err := cnnfag.Get(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cnnfag:", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintln(os.Stderr, "cnnfag:", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("%.0f (%s) as of %s\n", res.Score, res.Rating, res.Timestamp.Format(time.RFC3339))
	fmt.Printf("previous close %.0f · week ago %.0f · month ago %.0f · year ago %.0f\n",
		res.PreviousClose, res.OneWeekAgo, res.OneMonthAgo, res.OneYearAgo)
}
