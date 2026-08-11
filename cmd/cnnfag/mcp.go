package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"runtime/debug"
	"time"

	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
)

// The MCP stdio transport is JSON-RPC 2.0, one message per line. A server with
// a single argument-light tool needs only initialize, ping, tools/list and
// tools/call, which is small enough to implement on the standard library.
// Not implemented: cancellation, progress, and every server-to-client feature.

const toolName = "get_fear_and_greed"

// Protocol revisions this server knows. An initialize request asking for
// anything else is answered with the newest of these, as the spec directs.
var protocolVersions = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
}

const latestProtocolVersion = "2025-06-18"

type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

var toolDef = map[string]any{
	"name":        toolName,
	"description": "CNN's Fear & Greed index for the US stock market: current score (0-100) and rating, values for the previous close, week, month and year, and optionally about a year of daily history.",
	"inputSchema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"include_history": map[string]any{
				"type":        "boolean",
				"description": "Include about a year of daily scores. Off by default to keep the response small.",
			},
		},
		"additionalProperties": false,
	},
}

func serveMCP(r io.Reader, w io.Writer, fetch func(context.Context) (cnnfag.Result, error)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(w)

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if err := enc.Encode(rpcResponse{JSONRPC: "2.0", Error: &rpcError{-32700, "parse error"}}); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 || string(req.ID) == "null" {
			continue // a notification, nothing to answer
		}

		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "initialize":
			resp.Result = initializeResult(req.Params)
		case "ping":
			resp.Result = struct{}{}
		case "tools/list":
			resp.Result = map[string]any{"tools": []any{toolDef}}
		case "tools/call":
			resp.Result, resp.Error = callTool(req.Params, fetch)
		default:
			resp.Error = &rpcError{-32601, "method not found: " + req.Method}
		}

		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return sc.Err()
}

func initializeResult(params json.RawMessage) any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	json.Unmarshal(params, &p) // on any error the fallback below applies

	version := p.ProtocolVersion
	if !protocolVersions[version] {
		version = latestProtocolVersion
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      map[string]any{"name": "cnnfag", "version": buildVersion()},
	}
}

func callTool(params json.RawMessage, fetch func(context.Context) (cnnfag.Result, error)) (any, *rpcError) {
	var p struct {
		Name      string `json:"name"`
		Arguments struct {
			IncludeHistory bool `json:"include_history"`
		} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Name != toolName {
		return nil, &rpcError{-32602, "unknown tool"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := fetch(ctx)
	if err != nil {
		// A fetch failure is a tool-level error: the model should see it.
		return toolResult{Content: []textContent{{Type: "text", Text: err.Error()}}, IsError: true}, nil
	}
	if !p.Arguments.IncludeHistory {
		res.History = nil
	}

	text, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return nil, &rpcError{-32603, err.Error()}
	}
	return toolResult{Content: []textContent{{Type: "text", Text: string(text)}}}, nil
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "unknown"
}
