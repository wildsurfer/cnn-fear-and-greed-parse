// Command cnnfag prints CNN's Fear & Greed index as text or JSON, and can run
// a Model Context Protocol server exposing the index as a tool ("cnnfag mcp").
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cnnfag", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "print the full result, including history, as JSON")
	timeout := fs.Duration("timeout", 15*time.Second, "request timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	switch fs.Arg(0) {
	case "":
	case "mcp":
		if err := serveMCP(stdin, stdout, cnnfag.Get); err != nil {
			fmt.Fprintln(stderr, "cnnfag mcp:", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "cnnfag: unknown command %q, the only command is \"mcp\"\n", fs.Arg(0))
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	res, err := cnnfag.Get(ctx)
	if err != nil {
		fmt.Fprintln(stderr, "cnnfag:", err)
		return 1
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintln(stderr, "cnnfag:", err)
			return 1
		}
		return 0
	}

	fmt.Fprintf(stdout, "%.0f (%s) as of %s\n", res.Score, res.Rating, res.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(stdout, "previous close %.0f · week ago %.0f · month ago %.0f · year ago %.0f\n",
		res.PreviousClose, res.OneWeekAgo, res.OneMonthAgo, res.OneYearAgo)
	return 0
}
