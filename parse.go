package cnnfag

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const _url = "https://money.cnn.com/data/fear-and-greed/"

var ErrImgLoadNon200 = errors.New("image download failed, non 200 response code")
var ErrReadingBytes = errors.New("reading image bytes failed")
var ErrHTTPNon200 = errors.New("http status code isn't 200")
var ErrEmptyField = errors.New("at least one field is empty")

type ResultValueText struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type Result struct {
	ImageURL       string          `json:"imageUrl"`
	Now            ResultValueText `json:"now"`
	PreviousClose  ResultValueText `json:"previousClose"`
	OneWeekAgo     ResultValueText `json:"oneWeekAgo"`
	OneMonthAgo    ResultValueText `json:"oneMonthAgo"`
	OneYearAgo     ResultValueText `json:"oneYearAgo"`
	LastUpdateDate time.Time       `json:"lastUpdateDate"`
}

// GetImageBytes gets image in bytes.
func (r *Result) GetImageBytes() ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, r.ImageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do error: %w", err)
	}

	if res.StatusCode != 200 {
		return nil, ErrImgLoadNon200
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, ErrReadingBytes
	}

	_ = res.Body.Close()

	return bytes, nil
}

func _getLoc() *time.Location {
	loc, _ := time.LoadLocation("America/New_York")

	return loc
}

func _getGoqueryDocument() (*goquery.Document, error) {
	var emptyDoc goquery.Document

	res, err := _fetch()
	if err != nil {
		return &emptyDoc, err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return &emptyDoc, fmt.Errorf("goquery error: %w", err)
	}

	err = res.Body.Close()
	if err != nil {
		return &emptyDoc, fmt.Errorf("body closing error: %w", err)
	}

	return doc, nil
}

func _fetch() (res *http.Response, err error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, _url, nil)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}

	res, err = http.DefaultClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("http do error: %w", err)
	}

	if res.StatusCode != 200 {
		return nil, ErrHTTPNon200
	}

	return res, nil
}

func _parseImage(html string) string {
	re := regexp.MustCompile(`http://markets\.money\.cnn\.com/Marketsdata/uploadhandler/\w+\.png`)

	return re.FindString(html)
}

func _parseText(text string) (int, string) {
	re := regexp.MustCompile(`.+?(\d+)\s\((.+)\)`)
	sm := re.FindStringSubmatch(text)
	v, _ := strconv.ParseInt(sm[1], 10, 32)

	return int(v), sm[2]
}

func _parseDate(text string) time.Time {
	t, _ := time.Parse("Last updated Jan 2 at 3:04pm", text)

	today := time.Now().In(_getLoc())

	// As far as year isn't specified on money.cnn.com website we assume it to be the current one.
	t1 := time.Date(today.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, _getLoc())

	// We need to subtract one year if we parse "Dec 31" on January 1st
	if today.Before(t1) {
		return t1.AddDate(-1, 0, 0)
	}

	return t1
}

func _parse(doc *goquery.Document) (result Result, err error) {
	doc.Find("#fearGreedContainer .modContent").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		result.ImageURL = _parseImage(html)

		_populateResult(s, &result)

		text := s.Find("#needleAsOfDate").Text()
		result.LastUpdateDate = _parseDate(text)
	})

	if _isAnyFieldEmpty(result) {
		return result, ErrEmptyField
	}

	return result, err
}

func _isAnyFieldEmpty(r Result) bool {
	switch {
	case r.ImageURL == "":
		return true
	case _isFieldEmpty(r.Now):
		return true
	case _isFieldEmpty(r.PreviousClose):
		return true
	case _isFieldEmpty(r.OneWeekAgo):
		return true
	case _isFieldEmpty(r.OneMonthAgo):
		return true
	case _isFieldEmpty(r.OneYearAgo):
		return true
	case r.LastUpdateDate.IsZero():
		return true
	}

	return false
}

func _isFieldEmpty(r ResultValueText) bool {
	if r.Text == "" || r.Value == 0 {
		return true
	}

	return false
}

func _populateResult(s *goquery.Selection, result *Result) *goquery.Selection {
	return s.Find("ul li").Each(func(i int, ss *goquery.Selection) {
		switch i {
		case 0:
			result.Now.Value, result.Now.Text = _parseText(ss.Text())
		case 1:
			result.PreviousClose.Value, result.PreviousClose.Text = _parseText(ss.Text())
		case 2:
			result.OneWeekAgo.Value, result.OneWeekAgo.Text = _parseText(ss.Text())
		case 3:
			result.OneMonthAgo.Value, result.OneMonthAgo.Text = _parseText(ss.Text())
		case 4:
			result.OneYearAgo.Value, result.OneYearAgo.Text = _parseText(ss.Text())
		}
	})
}

// Parse is the only method you need to get data from CNN's Fear & Greed page.
func Parse() (Result, error) {
	doc, err := _getGoqueryDocument()
	if err != nil {
		return Result{}, err
	}

	result, err := _parse(doc)
	if err != nil {
		return Result{}, err
	}

	return result, nil
}
