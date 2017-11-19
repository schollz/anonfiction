package story

import (
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/pkg/errors"
	"github.com/schollz/storiesincognito/src/user"
	"github.com/schollz/versionedtext"
)

var DB string

func init() {
	DB = "stories.db"
}

type Story struct {
	ID         string    `storm:"unique"` // primary key, provided by client
	UserID     int       `storm:"index"`
	Date       time.Time `storm:"index"`
	Topic      string    `storm:"index"`
	Keywords   []string
	Paragraphs []template.HTML
	Content    versionedtext.VersionedText
	Published  bool `storm:"index"`
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
	log.Println(s.Published)
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

func Delete(id, apikey string) error {
	s, err := Get(id)
	if err != nil {
		return errors.Wrap(err, "story not deleted")
	}
	u, err := user.GetByAPIKey(apikey)
	if err != nil {
		return errors.Wrap(err, "story not deleted")
	}
	if s.UserID == u.ID || u.IsAdmin {
		db, err := storm.Open(DB)
		defer db.Close()
		if err != nil {
			return err
		}
		err = db.DeleteStruct(&s)
		if err != nil {
			return err
		}
	}
	return nil
}

// Update will create or update a story for a user
func Update(id, apikey, topic, content string, keywords []string, published bool) (s Story, err error) {
	s, errNew := Get(id)
	if errNew != nil {
		// create a new story since it doesn't exist
		return New(id, apikey, topic, content, keywords)
	}

	// story exists, update it
	u, err := user.GetByAPIKey(apikey)
	if err != nil {
		err = errors.New("must sign in to edit story")
		return
	}
	if !u.IsAdmin && s.Published {
		err = errors.New("cannot edit published story")
		return
	}
	if s.UserID != u.ID && !u.IsAdmin {
		err = errors.New("must sign in to edit story")
		return
	}
	s.Content.Update(content)
	s.Keywords = keywords
	for i, k := range s.Keywords {
		s.Keywords[i] = strings.ToLower(strings.TrimSpace(k))
	}
	s.Paragraphs = ConvertTrix(content)
	// only admin can publish
	if u.IsAdmin {
		log.Println("Publishing story", published)
		s.Published = published
	}
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.Save(&s)
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
