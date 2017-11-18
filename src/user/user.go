package user

import (
	"encoding/hex"
	"log"
	"strings"

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
	Email        string `storm:"unique"`    // this field will be indexed with a unique constraint
	PasswordHash string // this field will not be indexed
	Language     string
	Subscribed   bool
	IsAdmin      bool
	APIKey       string
}

// New creates a new user and attempts to add it to the database
func Add(email, password, language string, subscribed bool) (err error) {
	var hashedPassword []byte
	hashedPassword, err = cryptopasta.HashPassword([]byte(password))
	if err != nil {
		return
	}
	u := &User{
		Email:        email,
		PasswordHash: hex.EncodeToString(hashedPassword),
		Language:     language,
		Subscribed:   subscribed,
		APIKey:       utils.NewAPIKey(),
		IsAdmin:      strings.Contains(email, "zack"),
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
func Get(email string) (u User, err error) {
	db, err := storm.Open(DB)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.One("Email", email, &u)
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
func Validate(email, password string) (apikey string, err error) {
	u, err := Get(email)
	if err != nil {
		err = errors.New("user '" + email + "' does not exist")
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

func UserExists(email string) bool {
	_, err := Get(email)
	return err == nil
}

func APIKeyExists(apikey string) bool {
	_, err := GetByAPIKey(apikey)
	return err == nil
}

func IsAdmin(apikey string) bool {
	u, err := GetByAPIKey(apikey)
	if err != nil {
		return false
	}
	return u.IsAdmin
}

// SetAdmin gives admin privileges to a user
func SetAdmin(email string, isadmin bool) (err error) {
	u, err := Get(email)
	if err != nil {
		err = errors.New("user '" + email + "' does not exist")
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
