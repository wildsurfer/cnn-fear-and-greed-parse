// Package cnnfag fetches CNN's Fear & Greed index.
//
// CNN has no documented public API; this package uses the JSON endpoint that
// the https://www.cnn.com/markets/fear-and-greed page itself requests.
package cnnfag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// var, not const: tests point it at an httptest server.
var endpoint = "https://production.dataviz.cnn.io/index/fearandgreed/graphdata"

// ErrUnexpectedStatus is returned when CNN responds with a non-200 status.
// CNN answers 418 when a request is missing browser-like headers.
var ErrUnexpectedStatus = errors.New("unexpected http status")

// Point is one daily observation of the index.
type Point struct {
	Date   time.Time `json:"date"`
	Score  float64   `json:"score"`
	Rating string    `json:"rating"`
}

// Result holds the current state of the index and about a year of daily history.
type Result struct {
	Score         float64   `json:"score"`
	Rating        string    `json:"rating"`
	Timestamp     time.Time `json:"timestamp"`
	PreviousClose float64   `json:"previousClose"`
	OneWeekAgo    float64   `json:"oneWeekAgo"`
	OneMonthAgo   float64   `json:"oneMonthAgo"`
	OneYearAgo    float64   `json:"oneYearAgo"`
	History       []Point   `json:"history"`
}

type apiResponse struct {
	FearAndGreed struct {
		Score         float64   `json:"score"`
		Rating        string    `json:"rating"`
		Timestamp     time.Time `json:"timestamp"`
		PreviousClose float64   `json:"previous_close"`
		Previous1W    float64   `json:"previous_1_week"`
		Previous1M    float64   `json:"previous_1_month"`
		Previous1Y    float64   `json:"previous_1_year"`
	} `json:"fear_and_greed"`
	Historical struct {
		Data []struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Rating string  `json:"rating"`
		} `json:"data"`
	} `json:"fear_and_greed_historical"`
}

// Get fetches the current Fear & Greed index from CNN.
func Get(ctx context.Context) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, fmt.Errorf("building request: %w", err)
	}

	// The endpoint returns 418 unless both headers are present.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.cnn.com/markets/fear-and-greed")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("fetching fear and greed data: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("%w: %d", ErrUnexpectedStatus, res.StatusCode)
	}

	var raw apiResponse
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return Result{}, fmt.Errorf("decoding response: %w", err)
	}

	fg := raw.FearAndGreed
	result := Result{
		Score:         fg.Score,
		Rating:        fg.Rating,
		Timestamp:     fg.Timestamp,
		PreviousClose: fg.PreviousClose,
		OneWeekAgo:    fg.Previous1W,
		OneMonthAgo:   fg.Previous1M,
		OneYearAgo:    fg.Previous1Y,
		History:       make([]Point, 0, len(raw.Historical.Data)),
	}

	for _, d := range raw.Historical.Data {
		result.History = append(result.History, Point{
			Date:   time.UnixMilli(int64(d.X)).UTC(),
			Score:  d.Y,
			Rating: d.Rating,
		})
	}

	return result, nil
}
