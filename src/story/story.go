package story

import (
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
	ID      string `storm:"unique"` // primary key, provided by client
	UserID  int
	Date    time.Time
	Topic   string
	Content versionedtext.VersionedText
}

func New(id, apikey, topic, content string) (err error) {
	// first get user
	u, err := user.GetByAPIKey(apikey)
	if err != nil {
		return
	}

	s := Story{
		ID:      id,
		UserID:  u.ID,
		Date:    time.Now(),
		Topic:   topic,
		Content: versionedtext.NewVersionedText(content),
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
func Get(id, apikey string) (s Story, err error) {
	// first get user
	u, err := user.GetByAPIKey(apikey)
	if err != nil {
		return
	}

	// open story db
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}

	// get story
	query := db.Select(q.Eq("ID", id), q.Eq("UserID", u.ID))
	err = query.First(&s)
	return
}

// GetStory returns the content of the story for the API key
func GetStory(id, apikey string) (content string, err error) {
	s, err := Get(id, apikey)
	if err != nil {
		return
	}
	content = s.Content.GetCurrent()
	return
}

// Update will create or update a story for a user
func Update(id, apikey, topic, content string) (err error) {
	s, errNew := Get(id, apikey)
	if errNew != nil {
		// create a new story since it doesn't exist
		return New(id, apikey, topic, content)
	}

	// story exists, update it
	s.Content.Update(content)
	db, err := storm.Open(DB)
	defer db.Close()
	return db.Update(&s)
}
