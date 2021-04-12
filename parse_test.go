package cnn_fear_and_greed_parse

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/tkuchiki/faketime"
	"log"
	"os"
	"testing"
	"time"
)

func TestParseImage(t *testing.T) {
	html := "url(&#39;http://markets.money.cnn.com/Marketsdata/uploadhandler/z6f8f7d0az4c46c1b6d9644447a6d8829abaa17ece.png&#39;\""
	want := "http://markets.money.cnn.com/Marketsdata/uploadhandler/z6f8f7d0az4c46c1b6d9644447a6d8829abaa17ece.png"
	got := _parseImage(html)

	if got != want {
		t.Errorf("Url parse failed.\n\tWant: '%s',\n\t Got: '%s'", want, got)
	}
}

func TestParseText(t *testing.T) {
	gotValue, gotText := _parseText("Fear &amp; Greed Now: 44 (Fear)")
	assertValueAndText(
		t,
		"Now",
		44,
		"Fear",
		gotValue,
		gotText,
	)

	gotValue, gotText = _parseText("Fear &amp; Greed Previous Close: 52 (Neutral)")
	assertValueAndText(
		t,
		"PreviousClose",
		52,
		"Neutral",
		gotValue,
		gotText,
	)

	gotValue, gotText = _parseText("Fear &amp; Greed 1 Week Ago: 54 (Neutral)")
	assertValueAndText(
		t,
		"OneWeekAgo",
		54,
		"Neutral",
		gotValue,
		gotText,
	)

	gotValue, gotText = _parseText("Fear &amp; Greed 1 Month Ago: 48 (Neutral)")
	assertValueAndText(
		t,
		"OneMonthAgo",
		48,
		"Neutral",
		gotValue,
		gotText,
	)

	gotValue, gotText = _parseText("Fear &amp; Greed 1 Year Ago: 23 (Extreme Fear)")
	assertValueAndText(
		t,
		"OneYearAgo",
		23,
		"Extreme Fear",
		gotValue,
		gotText,
	)
}

// Covers the situation when today is the beginning of the year
// and last update was done at the end of previous year.
func TestParseDateEdgeCase(t *testing.T) {
	text := "Last updated Dec 31 at 4:59pm"

	f := faketime.NewFaketime(2021, time.Month(1), 01, 3, 28, 0, 0, _location)
	defer f.Undo()
	f.Do()

	want := time.Date(2020, time.Month(12), 31, 16, 59, 0, 0, _location)
	got := _parseDate(text)

	if got != want {
		t.Errorf("Date parse failed.\n\tWant: '%s',\n\t Got: '%s'", want, got)
	}
}

// Covers the situation when now and last updated time are exactly the same
func TestParseDateEdgeCase1(t *testing.T) {
	text := "Last updated Dec 31 at 11:59pm"

	f := faketime.NewFaketime(2020, time.Month(12), 31, 23, 59, 0, 0, _location)
	defer f.Undo()
	f.Do()

	want := time.Date(2020, time.Month(12), 31, 23, 59, 0, 0, _location)
	got := _parseDate(text)

	if got != want {
		t.Errorf("Date parse failed.\n\tWant: '%s',\n\t Got: '%s'", want, got)
	}
}

func TestParse(t *testing.T) {

	f := faketime.NewFaketime(2021, time.March, 29, 17, 60, 0, 0, _location)
	defer f.Undo()
	f.Do()

	doc := getGoqueryDocumentMock()
	result, _ := _parse(doc)
	wantLastUpdateDate := time.Date(2021, time.Month(3), 29, 16, 59, 0, 0, _location)

	want := Result{
		ImageUrl: "http://markets.money.cnn.com/Marketsdata/uploadhandler/z6f8f7d0az4c46c1b6d9644447a6d8829abaa17ece.png",
		Now: ResultValueText{
			Value: 44,
			Text:  "Fear",
		},
		PreviousClose: ResultValueText{
			Value: 52,
			Text:  "Neutral",
		},
		OneWeekAgo: ResultValueText{
			Value: 54,
			Text:  "Neutral",
		},
		OneMonthAgo: ResultValueText{
			Value: 48,
			Text:  "Neutral",
		},
		OneYearAgo: ResultValueText{
			Value: 23,
			Text:  "Extreme Fear",
		},
		//Last updated Mar 29 at 4:59pm
		LastUpdateDate: wantLastUpdateDate,
	}

	if want != result {
		t.Errorf("Result is invalid.\n\tWant: '%v',\n\t Got: '%v'", want, result)
	}

}

func assertValueAndText(t *testing.T, name string, wantValue int, wantText string, gotValue int, gotText string) {
	if gotValue != wantValue {
		t.Errorf("%s value is invalid.\n\tWant: '%d',\n\t Got: '%d'", name, wantValue, gotValue)
	}
	if gotText != wantText {
		t.Errorf("%s text is invalid.\n\tWant: '%s',\n\t Got: '%s'", name, wantText, gotText)
	}
}

func getGoqueryDocumentMock() *goquery.Document {
	f, err := os.Open("test/page.html")
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	return doc
}

//Uncomment below to update `test/page.html` file.
//Don't forget to update asserts in TestParse test.
//func UpdateTestAssets() {
//	res, err := http.Get(_url)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer res.Body.Close()
//
//	if res.StatusCode != 200 {
//		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
//	}
//
//	out, err := os.Create("test/page.html")
//	if err != nil {
//		log.Fatalf("Can't write to file: %s", err.Error())
//	}
//	defer out.Close()
//	io.Copy(out, res.Body)
//}
//
//func TestTmp(t *testing.T) {
//	UpdateTestAssets()
//}
