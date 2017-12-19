package user

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	os.Remove("user_test.db")
	DB = "user_test.db"
	err := Add("zack", "english", false)
	assert.Nil(t, err)
	err = Add("zack", "english", false)
	assert.NotNil(t, err)
}
