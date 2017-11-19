package topic

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/schollz/storiesincognito/src/story"
)

type Topic struct {
	Name            string
	Month           string
	Open            bool
	NumberOfStories int
}

func Load(filename string) (t []Topic, err error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &t)
	for i := range t {
		t[i].NumberOfStories = story.NumberOfStories(t[i].Name)
	}
	return
}

func IsClosed(filename string, topicName string) bool {
	topics, err := Load(filename)
	if err != nil {
		return true
	}
	for _, t := range topics {
		if strings.ToLower(topicName) == strings.ToLower(t.Name) {
			return !t.Open
		}
	}
	return true
}

// Next returns the next topic
func Next(filename string, topicName string) string {
	topics, err := Load(filename)
	if err != nil {
		return ""
	}
	for i, t := range topics {
		if i == 0 {
			continue
		}
		if strings.ToLower(topicName) == strings.ToLower(t.Name) {
			return topics[i-1].Name
		}
	}
	return ""
}

func Default(filename string, reading bool) (defaultTopic Topic, err error) {
	topics, err := Load(filename)
	if err != nil {
		return
	}
	for _, t := range topics {
		if t.Open == false {
			if reading == true {
				defaultTopic = t
			}
			break
		}
		defaultTopic = t
	}
	return
}

func Get(filename string, topicName string) (t Topic, err error) {
	topics, err := Load(filename)
	if err != nil {
		return
	}
	for _, topic := range topics {
		if strings.ToLower(topic.Name) == strings.ToLower(topicName) {
			t = topic
			t.NumberOfStories = story.NumberOfStories(t.Name)
			return
		}
	}
	err = errors.New("topic '" + topicName + "' not found")
	return
}
