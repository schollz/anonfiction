package topic

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopic(t *testing.T) {
	ioutil.WriteFile("topics.json", []byte(`[
		{
			"name":"Threats",
			"month": "January 2018",
			"open":true,
			"default":false
		},
		{
			"name":"Being Broke",
			"month": "December 2017",
			"open":true,
			"default":true
		},
		{
			"name":"Odd One Out",
			"month": "November 2017",
			"open":false,
			"default":false
		},
		{
			"name":"Dating",
			"month": "October 2017",
			"open":false,
			"default":false
		},
		{
			"name":"What Really Matters",
			"month": "September 2017",
			"open":false,
			"default":false
		}
	]`), 0644)
	topic, err := Load("topics.json")
	assert.Nil(t, err)
	assert.Equal(t, "Odd One Out", topic[2].Name)
	os.Remove("topics.json")
}
