package story

import (
	"errors"
	"html/template"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/schollz/storiesincognito/src/user"
	"github.com/schollz/versionedtext"
)

var DB string

func init() {
	DB = "stories.db"
}

type Story struct {
	ID         string `storm:"unique"` // primary key, provided by client
	UserID     int
	Date       time.Time
	Topic      string
	Keywords   []string
	Paragraphs []template.HTML
	Content    versionedtext.VersionedText
	Published  bool
}

func ListByUser(apikey string) (stories []Story, err error) {
	u, err := user.GetByAPIKey(apikey)
	if err != nil {
		err = errors.New("Incorrect API key")
		return
	}
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	query := db.Select(q.Eq("UserID", u.ID)).OrderBy("Date").Reverse()
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

func New(id, apikey, topic, content string, keywords []string) (s Story, err error) {
	// first get user
	u, err := user.GetByAPIKey(apikey)
	if err != nil {
		err = errors.New("Incorrect API key")
		return
	}

	s = Story{
		ID:         id,
		UserID:     u.ID,
		Date:       time.Now(),
		Topic:      topic,
		Content:    versionedtext.NewVersionedText(content),
		Keywords:   keywords,
		Paragraphs: ConvertTrix(content),
	}

	// open story db
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.Save(&s)
	return
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

// GetStory returns the content of the story for the API key
func GetStory(id, apikey string) (content string, err error) {
	s, err := Get(id)
	if err != nil {
		return
	}
	content = s.Content.GetCurrent()
	return
}

// Update will create or update a story for a user
func Update(id, apikey, topic, content string, keywords []string) (s Story, err error) {
	s, errNew := Get(id)
	if errNew != nil {
		// create a new story since it doesn't exist
		return New(id, apikey, topic, content, keywords)
	}

	// story exists, update it
	u, err := user.GetByAPIKey(apikey)
	if err != nil {
		return
	}
	if s.UserID != u.ID && !u.IsAdmin {
		err = errors.New("must sign in to edit this story")
		return
	}
	s.Content.Update(content)
	s.Keywords = keywords
	for i, k := range s.Keywords {
		s.Keywords[i] = strings.ToLower(strings.TrimSpace(k))
	}
	s.Paragraphs = ConvertTrix(content)
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.Update(&s)
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
		paragraphs[i] = template.HTML(strings.TrimSpace(p))
	}
	return
}
