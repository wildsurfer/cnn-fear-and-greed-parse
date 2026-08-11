package main

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
)

// fixtureTransport answers every request with the saved API response, so the
// CLI can be tested through cnnfag.HTTPClient without touching the network.
type fixtureTransport struct{ body []byte }

func (t fixtureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(t.body))),
		Header:     make(http.Header),
	}, nil
}

type errorTransport struct{}

func (errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, io.ErrUnexpectedEOF
}

func TestRun(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/graphdata.json")
	if err != nil {
		t.Fatal(err)
	}

	old := cnnfag.HTTPClient
	cnnfag.HTTPClient = &http.Client{Transport: fixtureTransport{fixture}}
	defer func() { cnnfag.HTTPClient = old }()

	var stdout, stderr strings.Builder
	if code := run(nil, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("run() = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "64 (greed)") {
		t.Errorf("text output: %q", stdout.String())
	}

	stdout.Reset()
	if code := run([]string{"-json"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("run(-json) = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"score": 64.3714285714286`) ||
		!strings.Contains(stdout.String(), `"junkBondDemand"`) {
		t.Errorf("json output: %q", stdout.String())
	}

	if code := run([]string{"bogus"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Errorf("run(bogus) = %d, want 2", code)
	}

	cnnfag.HTTPClient = &http.Client{Transport: errorTransport{}}
	if code := run(nil, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Errorf("run with failing transport = %d, want 1", code)
	}
	cnnfag.HTTPClient = &http.Client{Transport: fixtureTransport{fixture}}

	// The mcp subcommand wires stdin/stdout to serveMCP.
	stdout.Reset()
	in := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
	if code := run([]string{"mcp"}, strings.NewReader(in), &stdout, &stderr); code != 0 {
		t.Fatalf("run(mcp) = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"result":{}`) {
		t.Errorf("mcp ping output: %q", stdout.String())
	}
}
