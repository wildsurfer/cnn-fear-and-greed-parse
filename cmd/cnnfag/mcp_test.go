package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
)

func TestServeMCP(t *testing.T) {
	fetch := func(ctx context.Context) (cnnfag.Result, error) {
		return cnnfag.Result{
			Score:     43.71,
			Rating:    "fear",
			Timestamp: time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC),
			History: []cnnfag.Point{
				{Date: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC), Score: 60.2, Rating: "greed"},
			},
			MarketVolatility: cnnfag.Indicator{
				Score:   50,
				Rating:  "neutral",
				History: []cnnfag.Value{{Date: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC), Value: 17.5, Rating: "neutral"}},
			},
		}, nil
	}

	in := strings.Join([]string{
		`not json at all`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_fear_and_greed","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_fear_and_greed","arguments":{"include_history":true}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"no/such/method"}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"wrong_tool"}}`,
		`{"jsonrpc":"2.0","id":7,"method":"ping"}`,
	}, "\n") + "\n"

	var out strings.Builder
	if err := serveMCP(strings.NewReader(in), &out, fetch); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 9 {
		t.Fatalf("got %d responses, want 9 (the notification must not be answered):\n%s", len(lines), out.String())
	}

	var resp struct {
		ID     json.RawMessage `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code int `json:"code"`
		} `json:"error"`
	}

	// A line that is not JSON gets a parse error.
	mustUnmarshal(t, lines[0], &resp)
	if resp.Error == nil || resp.Error.Code != -32700 {
		t.Errorf("parse error response: %s", lines[0])
	}

	// An unsupported protocol version falls back to the latest we know.
	mustUnmarshal(t, lines[1], &resp)
	if !strings.Contains(string(resp.Result), `"protocolVersion":"2025-06-18"`) {
		t.Errorf("initialize fallback response: %s", lines[1])
	}

	// initialize echoes a supported protocol version.
	mustUnmarshal(t, lines[2], &resp)
	if !strings.Contains(string(resp.Result), `"protocolVersion":"2025-03-26"`) {
		t.Errorf("initialize response: %s", lines[2])
	}

	// tools/list advertises exactly our tool.
	mustUnmarshal(t, lines[3], &resp)
	var list struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	mustUnmarshal(t, string(resp.Result), &list)
	if len(list.Tools) != 1 || list.Tools[0].Name != "get_fear_and_greed" {
		t.Errorf("tools/list response: %s", lines[1])
	}

	// tools/call returns the score and the indicators, and omits every
	// history by default.
	mustUnmarshal(t, lines[4], &resp)
	if !strings.Contains(string(resp.Result), "43.71") ||
		!strings.Contains(string(resp.Result), "marketVolatility") ||
		strings.Contains(string(resp.Result), "history") ||
		strings.Contains(string(resp.Result), "17.5") {
		t.Errorf("tools/call without history: %s", lines[4])
	}

	// include_history brings the daily points in, for the index and the
	// indicators both.
	mustUnmarshal(t, lines[5], &resp)
	if !strings.Contains(string(resp.Result), "60.2") || !strings.Contains(string(resp.Result), "17.5") {
		t.Errorf("tools/call with history: %s", lines[5])
	}

	// Unknown methods get a JSON-RPC error.
	mustUnmarshal(t, lines[6], &resp)
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Errorf("unknown method response: %s", lines[6])
	}

	// Calling a tool we do not have is an invalid-params error.
	mustUnmarshal(t, lines[7], &resp)
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Errorf("unknown tool response: %s", lines[7])
	}

	// ping answers with an empty result.
	mustUnmarshal(t, lines[8], &resp)
	if string(resp.Result) != "{}" {
		t.Errorf("ping response: %s", lines[8])
	}
}

func mustUnmarshal(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("unmarshal %q: %v", data, err)
	}
}
