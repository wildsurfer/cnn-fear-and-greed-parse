package cnnfag

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Tests live in package cnnfag so they can swap the unexported endpoint
// variable; for the same reason they must not run in parallel.

func TestGet(t *testing.T) {
	fixture, err := os.ReadFile("testdata/graphdata.json")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" || r.Header.Get("Referer") == "" {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	old := endpoint
	endpoint = srv.URL
	defer func() { endpoint = old }()

	result, err := Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Score != 63.6857142857143 {
		t.Errorf("Score = %v, want 63.6857142857143", result.Score)
	}
	if result.Rating != "greed" {
		t.Errorf("Rating = %q, want %q", result.Rating, "greed")
	}
	wantTS := time.Date(2026, 8, 10, 11, 53, 1, 0, time.UTC)
	if !result.Timestamp.Equal(wantTS) {
		t.Errorf("Timestamp = %v, want %v", result.Timestamp, wantTS)
	}
	if result.PreviousClose != 63.6857142857143 {
		t.Errorf("PreviousClose = %v, want 63.6857142857143", result.PreviousClose)
	}
	if result.OneWeekAgo != 50.7428571428571 {
		t.Errorf("OneWeekAgo = %v, want 50.7428571428571", result.OneWeekAgo)
	}
	if result.OneMonthAgo != 46.82857142857143 {
		t.Errorf("OneMonthAgo = %v, want 46.82857142857143", result.OneMonthAgo)
	}
	if result.OneYearAgo != 58.37142857142857 {
		t.Errorf("OneYearAgo = %v, want 58.37142857142857", result.OneYearAgo)
	}
	if len(result.History) != 3 {
		t.Fatalf("len(History) = %d, want 3", len(result.History))
	}
	first := result.History[0]
	wantDate := time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC)
	if !first.Date.Equal(wantDate) {
		t.Errorf("History[0].Date = %v, want %v", first.Date, wantDate)
	}
	if first.Score != 57.628571428571426 {
		t.Errorf("History[0].Score = %v, want 57.628571428571426", first.Score)
	}
	if first.Rating != "greed" {
		t.Errorf("History[0].Rating = %q, want %q", first.Rating, "greed")
	}
}

func TestGetUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	old := endpoint
	endpoint = srv.URL
	defer func() { endpoint = old }()

	_, err := Get(context.Background())
	if !errors.Is(err, ErrUnexpectedStatus) {
		t.Fatalf("err = %v, want ErrUnexpectedStatus", err)
	}
}

func TestGetLive(t *testing.T) {
	if os.Getenv("CNNFAG_LIVE") != "1" {
		t.Skip("set CNNFAG_LIVE=1 to run the live test")
	}

	result, err := Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score <= 0 || result.Score > 100 {
		t.Errorf("Score = %v, want in (0, 100]", result.Score)
	}
	if result.Rating == "" {
		t.Error("Rating is empty")
	}
	if len(result.History) == 0 {
		t.Error("History is empty")
	}
}
