package user

import (
	"log"
	"time"

	"github.com/asdine/storm"
	"github.com/pkg/errors"
	"github.com/schollz/storiesincognito/src/utils"
)

var DB string

func init() {
	DB = "users.db"
	Add("anonymous", "english", false)
}

type User struct {
	ID         string `storm:"unique"` // primary key
	Email      string `storm:"unique"` // this field will be indexed with a unique constraint
	Language   string
	Subscribed bool
	Joined     time.Time
}

func AnonymousUserID() string {
	userID, err := GetID("anonymous")
	if err != nil {
		log.Fatal(err)
	}
	return userID
}

// New creates a new user and attempts to add it to the database
func Add(email, language string, subscribed bool) (err error) {
	u := &User{
		ID:         utils.NewAPIKey(),
		Email:      email,
		Language:   language,
		Subscribed: subscribed,
		Joined:     time.Now(),
	}
	log.Println("opening db to Add")
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.Save(u)
	log.Println("saved user " + email)
	if err == storm.ErrAlreadyExists {
		err = errors.Wrap(err, "'"+email+"' is taken")
	}
	return
}

// Get returns the User object for the specified email
func Get(id string) (u User, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.One("ID", id, &u)
	return
}

func GetID(email string) (userID string, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	var u User
	err = db.One("Email", email, &u)
	if err != nil {
		return
	}
	userID = u.ID
	return
}

func UserExists(id string) bool {
	_, err := Get(id)
	return err == nil
}

func All() (u []User, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		err = errors.Wrap(err, "problem opening DB")
		return
	}
	err = db.AllByIndex("ID", &u)
	if err != nil {
		err = errors.Wrap(err, "problem getting all by ID")
	}
	return
}
