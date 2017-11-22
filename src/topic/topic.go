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
	Description     string
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

func Active(filename string) (newT []Topic, err error) {
	t, err := Load(filename)
	if err != nil {
		return
	}
	newT = make([]Topic, len(t))
	newTI := 0
	for _, topic := range t {
		if topic.Open {
			newT[newTI] = topic
			newTI++
		}
	}
	newT = newT[:newTI]
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
	defaultTopic = topics[0]
	for _, t := range topics {
		if t.Open == true {
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
