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
			"name": "Wild Animals",
			"description": "Getting in touch with nature and the creatues around us.",
			"month": "2018-02-10T23:00:00Z",
			"open": true
		},
		{
			"name": "First Time",
			"description": "The first time I did X.",
			"date": "2018-02-10T23:00:00Z",
			"open": true
		},
		{
			"name": "Just The Two Of Us",
			"description": "Stories about family, friends, enemies, or lovers.",
			"month": "2018-03-10T23:00:00Z",
			"open": true
		},
		{
			"name": "Dreams",
			"description": "Aspirations, hopes, or just days where you think you are dreaming.",
			"month": "2018-04-10T23:00:00Z",
			"open": true
		},
	
		{
			"name": "I Laughed",
			"description": "Things that made me laugh, whether appropriate or not.",
			"month": "2018-05-10T23:00:00Z",
			"open": true
		}
	]`), 0644)
	topic, err := Load("topics.json")
	assert.Nil(t, err)
	assert.Equal(t, "Just The Two Of Us", topic[2].Name)
	assert.Equal(t, "2018-02-10 23:00:00 +0000 UTC", topic[1].Date.String())
	os.Remove("topics.json")
}
