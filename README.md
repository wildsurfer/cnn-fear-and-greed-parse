# cnn-fear-and-greed-parse [![Go Reference](https://pkg.go.dev/badge/github.com/wildsurfer/cnn-fear-and-greed-parse.svg)](https://pkg.go.dev/github.com/wildsurfer/cnn-fear-and-greed-parse)
Golang package to fetch and parse data from https://money.cnn.com/data/fear-and-greed/

## Usage example:
```go
package main

import (
	"encoding/json"
	"fmt"
	CNNParse "github.com/wildsurfer/cnn-fear-and-greed-parse"
)

func main() {
	result, err := CNNParse.Parse()
	if err != nil {
		panic(err)
	}

	b, _ := json.Marshal(result)
	fmt.Println(string(b))
}
```
### Output
```json
{
  "imageUrl":"http://markets.money.cnn.com/Marketsdata/uploadhandler/z748f7c0aza9d607b00bf84c6a8b0283e2745ded10.png",
  "now":{
    "value":44,
    "text":"Fear"
  },
  "previousClose":{
    "value":44,
    "text":"Fear"
  },
  "oneWeekAgo":{
    "value":49,
    "text":"Neutral"
  },
  "oneMonthAgo":{
    "value":48,
    "text":"Neutral"
  },
  "oneYearAgo":{
    "value":25,
    "text":"Extreme Fear"
  },
  "lastUpdateDate":"2020-03-30T15:50:00-05:00"
}
```
