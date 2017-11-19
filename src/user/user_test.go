package user

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	os.Remove("user_test.db")
	DB = "user_test.db"
	err := Add("zack", "pass123", "english", false)
	assert.Nil(t, err)
	err = Add("zack", "pass123", "english", false)
	assert.NotNil(t, err)
	var apikey string
	apikey, err = Validate("zack", "pass123")
	assert.Nil(t, err)
	assert.Equal(t, 10, len(apikey))
	apikey, err = Validate("zack", "pass122")
	assert.NotNil(t, err)
	apikey, err = Validate("zafck", "pass122")
	assert.NotNil(t, err)

	// test admin rights
	Add("jlkjklj", "pass123", "english", false)
	u, _ := Get("jlkjklj")
	assert.Equal(t, false, u.IsAdmin)
	err = SetAdmin("jlkjklj", true)
	assert.Nil(t, err)
	u, _ = Get("jlkjklj")
	assert.Equal(t, true, u.IsAdmin)

	// test re-adding
	Add("bob1", "pass123", "english", false)
	u, _ = Get("bob1")
	err = Add("bob1", "pass234", "english", false)
	assert.NotNil(t, err)
	u2, _ := Get("bob1")
	assert.Equal(t, u2.PasswordHash, u.PasswordHash)
}
