package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"
)

type Quote struct {
	ID   int    `json:"ID"`
	Text string `json:"Text"`
	Name string `json:"Name"`
}

func RandomQuote() string {
	bQuotes, err := ioutil.ReadFile("./static/story_quotes.json")
	if err != nil {
		panic(err)
	}
	var quotes []Quote
	err = json.Unmarshal(bQuotes, &quotes)
	if err != nil {
		panic(err)
	}
	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s) // initialize local pseudorandom generator
	quoteString := ""
	for i := 1; i < 10; i++ {
		q := quotes[r.Intn(len(quotes))]
		quoteString = fmt.Sprintf("“%s” - %s", q.Text, q.Name)
		if len(quoteString) < 80 {
			break
		}
	}
	return quoteString

}
