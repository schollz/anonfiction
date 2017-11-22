package story

import (
	"html/template"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/pkg/errors"
	"github.com/schollz/storiesincognito/src/utils"
	"github.com/schollz/versionedtext"
)

var DB string

func init() {
	DB = "stories.db"
}

type Story struct {
	ID          string    `storm:"unique"` // primary key, provided by client
	UserID      string    `storm:"index"`
	Date        time.Time `storm:"index"`
	Topic       string    `storm:"index"`
	Keywords    []string
	Paragraphs  []template.HTML
	Content     versionedtext.VersionedText
	Published   bool `storm:"index"`
	Description string
}

func ListByKeyword(keyword string) (stories []Story, err error) {
	allStories, err := All()
	if err != nil {
		return
	}
	stories = make([]Story, len(allStories)+1)
	storyI := 0
	for _, s := range allStories {
		if stringInSlice(keyword, s.Keywords) {
			stories[storyI] = s
			storyI++
		}
	}
	stories = stories[:storyI]
	return
}

func ListByUser(userID string) (stories []Story, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	query := db.Select(q.Eq("UserID", userID)).OrderBy("Date").Reverse()
	err = query.Find(&stories)
	return
}

func ListByTopic(topic string) (stories []Story, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	query := db.Select(q.Eq("Topic", topic), q.Eq("Published", true)).OrderBy("Date")
	err = query.Find(&stories)
	return
}

func NumberOfStories(topic string) int {
	s, _ := ListByTopic(topic)
	return len(s)
}

func New(userID, topic, content, description string, keywords []string) (s Story) {
	return Story{
		ID:          utils.NewAPIKey(),
		UserID:      userID,
		Date:        time.Now(),
		Topic:       topic,
		Content:     versionedtext.NewVersionedText(content),
		Description: description,
		Keywords:    keywords,
		Paragraphs:  ConvertTrix(content),
	}
}

// Get returns the story for the specified API key
func Get(id string) (s Story, err error) {
	// open story db
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}

	// get story
	query := db.Select(q.Eq("ID", id))
	err = query.First(&s)
	return
}

func (s Story) Save() (err error) {
	// open story db
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	s.Paragraphs = ConvertTrix(s.Content.GetCurrent())
	err = db.Save(&s)
	return
}

func (s Story) Delete() (err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return err
	}
	err = db.DeleteStruct(&s)
	return
}

func All() (s []Story, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		err = errors.Wrap(err, "problem opening DB")
		return
	}
	err = db.AllByIndex("Date", &s)
	if err != nil {
		err = errors.Wrap(err, "problem getting all by date")
	}
	return
}

func ConvertTrix(s string) (paragraphs []template.HTML) {
	filteredContent := strings.Replace(s, "<div>", "", -1)
	filteredContent = strings.Replace(filteredContent, "</div>", "", -1)
	filteredContent = strings.Replace(filteredContent, "<br><br>", "<break>", -1)
	filteredContent = strings.Replace(filteredContent, "<br>", " ", -1)
	paragraphText := strings.Split(filteredContent, "<break>")
	paragraphs = make([]template.HTML, len(paragraphText))
	for i, p := range paragraphText {
		if i == 0 {
			p = `<span class="leader f4">` + strings.Replace(p, ". ", "</span>. ", 1)

		}
		paragraphs[i] = template.HTML(strings.TrimSpace(p))
	}
	return
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
