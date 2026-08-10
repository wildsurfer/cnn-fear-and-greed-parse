package cnnfag_test

import (
	"context"
	"fmt"

	cnnfag "github.com/wildsurfer/cnn-fear-and-greed-parse/v2"
)

func ExampleGet() {
	result, err := cnnfag.Get(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("%.0f (%s)\n", result.Score, result.Rating)
	fmt.Printf("history: %d daily points\n", len(result.History))
}
