package user

import (
	"encoding/hex"
	"log"

	"github.com/asdine/storm"
	"github.com/gtank/cryptopasta"
	"github.com/pkg/errors"
	"github.com/schollz/storiesincognito/src/utils"
)

var DB string

func init() {
	DB = "users.db"
}

type User struct {
	ID           int    `storm:"increment"` // primary key
	Username     string `storm:"unique"`    // this field will be indexed with a unique constraint
	PasswordHash string // this field will not be indexed
	Language     string
	Subscribed   bool
	IsAdmin      bool
	APIKey       string
}

// New creates a new user and attempts to add it to the database
func Add(username, password, language string, subscribed bool) (err error) {
	var hashedPassword []byte
	hashedPassword, err = cryptopasta.HashPassword([]byte(password))
	if err != nil {
		return
	}
	u := &User{
		Username:     username,
		PasswordHash: hex.EncodeToString(hashedPassword),
		Language:     language,
		Subscribed:   subscribed,
		APIKey:       utils.NewAPIKey(),
	}
	log.Println("opening db to Add")
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.Save(u)
	log.Println("saved user " + username)
	if err == storm.ErrAlreadyExists {
		err = errors.Wrap(err, "'"+username+"' is taken")
	}
	return
}

// Get returns the User object for the specified username
func Get(username string) (u User, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.One("Username", username, &u)
	return
}

// GetByAPIKey returns the User object for the specified API key
func GetByAPIKey(apikey string) (u User, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.One("APIKey", apikey, &u)
	return
}

// Validate checks the password for the specified user, and if successful, returns the APIKey
func Validate(username, password string) (apikey string, err error) {
	u, err := Get(username)
	if err != nil {
		err = errors.New("user '" + username + "' does not exist")
		return
	}

	passwordHashBytes, err := hex.DecodeString(u.PasswordHash)
	if err != nil {
		err = errors.Wrap(err, "problem decoding password hash")
		return
	}
	err = cryptopasta.CheckPasswordHash(passwordHashBytes, []byte(password))
	if err != nil {
		err = errors.New("incorrect passphrase")
	}
	apikey = u.APIKey
	return
}

func UserExists(username string) bool {
	_, err := Get(username)
	return err == nil
}

// SetAdmin gives admin privileges to a user
func SetAdmin(username string, isadmin bool) (err error) {
	u, err := Get(username)
	if err != nil {
		err = errors.New("user '" + username + "' does not exist")
		return
	}

	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	u.IsAdmin = isadmin
	err = db.Update(&u)
	return
}
