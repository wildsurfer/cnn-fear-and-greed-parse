package cnnfag_test

import (
	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	result, err := cnnfag.Parse()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ImageURL == "" {
		t.Errorf("ImageURL is empty")
	}

	if isEmpty(t, result.Now) {
		t.Errorf("Now is empty")
	}

	if isEmpty(t, result.PreviousClose) {
		t.Errorf("PreviousClose is empty")
	}

	if isEmpty(t, result.OneWeekAgo) {
		t.Errorf("OneWeekAgo is empty")
	}

	if isEmpty(t, result.OneMonthAgo) {
		t.Errorf("OneMonthAgo is empty")
	}

	if isEmpty(t, result.OneYearAgo) {
		t.Errorf("OneYearAgo is empty")
	}

	if result.LastUpdateDate.IsZero() {
		t.Errorf("LastUpdateDate is empty")
	}
}

func TestResult_GetImageBytes(t *testing.T) {
	t.Parallel()

	result, _ := cnnfag.Parse()

	bytes, err := result.GetImageBytes()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if l := len(bytes); l < 10 {
		t.Errorf("Image bytes size '%d' is suspicious", l)
	}
}

func isEmpty(t *testing.T, result cnnfag.ResultValueText) bool {
	t.Helper()

	if result.Text == "" || result.Value == 0 {
		return true
	}

	return false
}
