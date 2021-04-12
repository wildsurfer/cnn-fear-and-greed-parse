package cnn_fear_and_greed_parse

import (
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

// "All times are ET" as said on money.cnn.com website in footer
var _location *time.Location

type ResultValueText struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type Result struct {
	ImageUrl       string          `json:"imageUrl"`
	Now            ResultValueText `json:"now"`
	PreviousClose  ResultValueText `json:"previousClose"`
	OneWeekAgo     ResultValueText `json:"oneWeekAgo"`
	OneMonthAgo    ResultValueText `json:"oneMonthAgo"`
	OneYearAgo     ResultValueText `json:"oneYearAgo"`
	LastUpdateDate time.Time       `json:"lastUpdateDate"`
}

func init() {
	_location, _ = time.LoadLocation("America/New_York")
}

// Use it to get image bytes
func (r *Result) GetImageBytes() ([]byte, error) {
	response, err := http.Get(r.ImageUrl)

	if err != nil {
		return nil, errors.New("image download failed")
	} else if response.StatusCode != 200 {
		return nil, errors.New("image download failed, non 200 response code")
	}

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("reading image bytes failed")
	}

	err = response.Body.Close()
	return bytes, err
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

	err = res.Body.Close()
	return doc, err
}

func _fetch() (*http.Response, error) {
	res, err := http.Get(_url)
	if err != nil {
		return nil, fmt.Errorf("http.Get() error: %v", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("http status code error: %d %s", res.StatusCode, res.Status)
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

	today := time.Now().In(_location)

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
		result.LastUpdateDate = _parseDate(text)
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
