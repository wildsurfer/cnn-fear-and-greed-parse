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

// HTTPClient is the client Get uses. Replace it to set a timeout, a proxy or
// a custom transport.
var HTTPClient = http.DefaultClient

// ErrUnexpectedStatus is returned when CNN responds with a non-200 status.
// CNN answers 418 when a request is missing browser-like headers.
var ErrUnexpectedStatus = errors.New("unexpected http status")

// ErrEmptyResult is returned when CNN answers 200 but the payload carries no
// index data, which most likely means the API schema changed.
var ErrEmptyResult = errors.New("empty result, CNN may have changed the API schema")

// Point is one daily observation of the index.
type Point struct {
	// Date is the start of the trading day in UTC. The newest point carries
	// the exact time of CNN's latest update instead.
	Date   time.Time `json:"date"`
	Score  float64   `json:"score"`
	Rating string    `json:"rating"`
}

// Value is one daily observation of an indicator's underlying series.
type Value struct {
	Date time.Time `json:"date"`
	// Value is the raw measurement (an index level, a ratio, a spread), not
	// a 0-100 score.
	Value  float64 `json:"value"`
	Rating string  `json:"rating"`
}

// Indicator is one of the seven components CNN combines into the index.
type Indicator struct {
	// Score is CNN's 0-100 normalization of the indicator.
	Score     float64   `json:"score"`
	Rating    string    `json:"rating"`
	Timestamp time.Time `json:"timestamp"`
	// History holds about a year of daily raw values, oldest first.
	History []Value `json:"history,omitempty"`
}

// Result holds the current state of the index and about a year of daily history.
//
// Scores use CNN's 0–100 scale, where 0 is extreme fear and 100 is extreme
// greed. Ratings are CNN's text labels for score bands: "extreme fear",
// "fear", "neutral", "greed", "extreme greed".
type Result struct {
	Score  float64 `json:"score"`
	Rating string  `json:"rating"`
	// Timestamp is when CNN last updated the score.
	Timestamp     time.Time `json:"timestamp"`
	PreviousClose float64   `json:"previousClose"`
	OneWeekAgo    float64   `json:"oneWeekAgo"`
	OneMonthAgo   float64   `json:"oneMonthAgo"`
	OneYearAgo    float64   `json:"oneYearAgo"`
	// History holds daily scores for roughly the past year, oldest first.
	History []Point `json:"history,omitempty"`

	// The seven component indicators. CNN's API also serves 125-day and
	// 50-day moving-average variants of momentum and volatility; those are
	// chart overlays with the same scores and are not exposed here.
	MarketMomentum     Indicator `json:"marketMomentum"`
	StockPriceStrength Indicator `json:"stockPriceStrength"`
	StockPriceBreadth  Indicator `json:"stockPriceBreadth"`
	PutCallOptions     Indicator `json:"putCallOptions"`
	MarketVolatility   Indicator `json:"marketVolatility"`
	JunkBondDemand     Indicator `json:"junkBondDemand"`
	SafeHavenDemand    Indicator `json:"safeHavenDemand"`
}

type apiSeries struct {
	// Timestamp is epoch milliseconds here; only fear_and_greed uses RFC 3339.
	Timestamp float64 `json:"timestamp"`
	Score     float64 `json:"score"`
	Rating    string  `json:"rating"`
	Data      []struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Rating string  `json:"rating"`
	} `json:"data"`
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
	Historical         apiSeries `json:"fear_and_greed_historical"`
	MarketMomentum     apiSeries `json:"market_momentum_sp500"`
	StockPriceStrength apiSeries `json:"stock_price_strength"`
	StockPriceBreadth  apiSeries `json:"stock_price_breadth"`
	PutCallOptions     apiSeries `json:"put_call_options"`
	MarketVolatility   apiSeries `json:"market_volatility_vix"`
	JunkBondDemand     apiSeries `json:"junk_bond_demand"`
	SafeHavenDemand    apiSeries `json:"safe_haven_demand"`
}

func toIndicator(s apiSeries) Indicator {
	ind := Indicator{
		Score:     s.Score,
		Rating:    s.Rating,
		Timestamp: time.UnixMilli(int64(s.Timestamp)).UTC(),
		History:   make([]Value, 0, len(s.Data)),
	}
	for _, d := range s.Data {
		ind.History = append(ind.History, Value{
			Date:   time.UnixMilli(int64(d.X)).UTC(),
			Value:  d.Y,
			Rating: d.Rating,
		})
	}
	return ind
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

	res, err := HTTPClient.Do(req)
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
	if fg.Rating == "" || fg.Timestamp.IsZero() {
		return Result{}, ErrEmptyResult
	}

	result := Result{
		Score:         fg.Score,
		Rating:        fg.Rating,
		Timestamp:     fg.Timestamp,
		PreviousClose: fg.PreviousClose,
		OneWeekAgo:    fg.Previous1W,
		OneMonthAgo:   fg.Previous1M,
		OneYearAgo:    fg.Previous1Y,
		History:       make([]Point, 0, len(raw.Historical.Data)),

		MarketMomentum:     toIndicator(raw.MarketMomentum),
		StockPriceStrength: toIndicator(raw.StockPriceStrength),
		StockPriceBreadth:  toIndicator(raw.StockPriceBreadth),
		PutCallOptions:     toIndicator(raw.PutCallOptions),
		MarketVolatility:   toIndicator(raw.MarketVolatility),
		JunkBondDemand:     toIndicator(raw.JunkBondDemand),
		SafeHavenDemand:    toIndicator(raw.SafeHavenDemand),
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
