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

	if result.Score != 64.3714285714286 {
		t.Errorf("Score = %v, want 64.3714285714286", result.Score)
	}
	if result.Rating != "greed" {
		t.Errorf("Rating = %q, want %q", result.Rating, "greed")
	}
	wantTS := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	if !result.Timestamp.Equal(wantTS) {
		t.Errorf("Timestamp = %v, want %v", result.Timestamp, wantTS)
	}
	if result.PreviousClose != 64.3714285714286 {
		t.Errorf("PreviousClose = %v, want 64.3714285714286", result.PreviousClose)
	}
	if result.OneWeekAgo != 59.9714285714286 {
		t.Errorf("OneWeekAgo = %v, want 59.9714285714286", result.OneWeekAgo)
	}
	if result.OneMonthAgo != 46.82857142857143 {
		t.Errorf("OneMonthAgo = %v, want 46.82857142857143", result.OneMonthAgo)
	}
	if result.OneYearAgo != 57.628571428571426 {
		t.Errorf("OneYearAgo = %v, want 57.628571428571426", result.OneYearAgo)
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

	jb := result.JunkBondDemand
	if jb.Score != 98.6 {
		t.Errorf("JunkBondDemand.Score = %v, want 98.6", jb.Score)
	}
	if jb.Rating != "extreme greed" {
		t.Errorf("JunkBondDemand.Rating = %q, want %q", jb.Rating, "extreme greed")
	}
	wantJBTS := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	if !jb.Timestamp.Equal(wantJBTS) {
		t.Errorf("JunkBondDemand.Timestamp = %v, want %v", jb.Timestamp, wantJBTS)
	}
	if len(jb.History) != 3 {
		t.Fatalf("len(JunkBondDemand.History) = %d, want 3", len(jb.History))
	}
	if jb.History[0].Value != 1.3148745353159097 {
		t.Errorf("JunkBondDemand.History[0].Value = %v, want 1.3148745353159097", jb.History[0].Value)
	}
	if jb.History[0].Rating != "extreme fear" {
		t.Errorf("JunkBondDemand.History[0].Rating = %q, want %q", jb.History[0].Rating, "extreme fear")
	}

	// Indicator histories carry raw values, momentum's is the S&P level.
	if result.MarketMomentum.History[0].Value != 6373.45 {
		t.Errorf("MarketMomentum.History[0].Value = %v, want 6373.45", result.MarketMomentum.History[0].Value)
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

func TestGetEmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	old := endpoint
	endpoint = srv.URL
	defer func() { endpoint = old }()

	_, err := Get(context.Background())
	if !errors.Is(err, ErrEmptyResult) {
		t.Fatalf("err = %v, want ErrEmptyResult", err)
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
	if result.MarketVolatility.Score <= 0 || result.MarketVolatility.Score > 100 {
		t.Errorf("MarketVolatility.Score = %v, want in (0, 100]", result.MarketVolatility.Score)
	}
	if len(result.MarketVolatility.History) == 0 {
		t.Error("MarketVolatility.History is empty")
	}
}
