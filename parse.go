package cnn_fear_and_greed_parse

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const _url = "https://money.cnn.com/data/fear-and-greed/"

// "All times are ET" as said on money.cnn.com website in footer
var _location *time.Location

type ResultNow struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type ResultPreviousClose struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type ResultOneWeekAgo struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type ResultOneMonthAgo struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type ResultOneYearAgo struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type Result struct {
	ImageUrl       string              `json:"image_url"`
	Now            ResultNow           `json:"now"`
	PreviousClose  ResultPreviousClose `json:"previous_close"`
	OneWeekAgo     ResultOneWeekAgo    `json:"one_week_ago"`
	OneMonthAgo    ResultOneMonthAgo   `json:"one_month_ago"`
	OneYearAgo     ResultOneYearAgo    `json:"one_year_ago"`
	LastUpdateDate time.Time           `json:"last_update_date"`
}

func init() {
	_location, _ = time.LoadLocation("EST")
}

func _getGoqueryDocument() (*goquery.Document, error) {
	emptyDoc := goquery.Document{}

	res, err := _fetch()
	if err != nil {
		return &emptyDoc, err
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return &emptyDoc, fmt.Errorf("goquery error: %v", err)
	}

	return doc, nil
}

func _fetch() (*http.Response, error) {
	res, err := http.Get(_url)
	if err != nil {
		return nil, fmt.Errorf("http.Get() error: %v", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("http status code error: %d %s", res.StatusCode, res.Status)
	}
	defer res.Body.Close()
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

func _parseDate(text string, today time.Time) time.Time {
	t, _ := time.Parse("Last updated Jan 2 at 3:04pm", text)

	// As far as year isn't specified on money.cnn.com website we assume it to be the current one.
	t1 := time.Date(today.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, _location)

	// We need to subtract one year if we parse "Dec 31" on January 1st
	if today.Before(t1) {
		return t1.AddDate(-1, 0, 0)
	} else {
		return t1
	}
}

func _parse(doc *goquery.Document) (Result, error) {
	result := Result{}
	doc.Find("#fearGreedContainer .modContent").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		result.ImageUrl = _parseImage(html)

		s.Find("ul li").Each(func(i int, ss *goquery.Selection) {
			switch i {
			case 0:
				result.Now.Value, result.Now.Text = _parseText(ss.Text())
				break
			case 1:
				result.PreviousClose.Value, result.PreviousClose.Text = _parseText(ss.Text())
				break
			case 2:
				result.OneWeekAgo.Value, result.OneWeekAgo.Text = _parseText(ss.Text())
				break
			case 3:
				result.OneMonthAgo.Value, result.OneMonthAgo.Text = _parseText(ss.Text())
				break
			case 4:
				result.OneYearAgo.Value, result.OneYearAgo.Text = _parseText(ss.Text())
				break
			}
		})

		text := s.Find("#needleAsOfDate").Text()
		result.LastUpdateDate = _parseDate(text, time.Now().In(_location))
	})

	fieldIsEmpty := false

	if result.ImageUrl == "" ||
		result.PreviousClose.Value == 0 ||
		result.PreviousClose.Text == "" ||
		result.Now.Value == 0 ||
		result.Now.Text == "" ||
		result.OneWeekAgo.Value == 0 ||
		result.OneWeekAgo.Text == "" ||
		result.OneMonthAgo.Value == 0 ||
		result.OneMonthAgo.Text == "" ||
		result.OneYearAgo.Value == 0 ||
		result.OneYearAgo.Text == "" ||
		result.LastUpdateDate.IsZero() {
		fieldIsEmpty = true
	}

	if fieldIsEmpty == true {
		return result, errors.New("at least one field is empty")
	}

	return result, nil
}

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
