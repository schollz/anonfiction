package story

import (
	"os"
	"testing"

	"github.com/schollz/storiesincognito/src/user"
	"github.com/stretchr/testify/assert"
)

func TestStory(t *testing.T) {
	os.Remove("user_test.db")
	os.Remove("story_test.db")
	user.DB = "user_test.db"
	DB = "story_test.db"
	err := user.Add("zack", "123", "english", false)
	assert.Nil(t, err)

	apikey, err := user.Validate("zack", "123")
	assert.Nil(t, err)
	err = Update("story0", apikey, "being", "to be or not to be, that is the question")
	assert.Nil(t, err)

	content, err := GetStory("story0", apikey)
	assert.Nil(t, err)
	assert.Equal(t, "to be or not to be, that is the question", content)

	err = Update("story0", apikey, "being", "to be or not to be, that is the question?")
	assert.Nil(t, err)

	content, err = GetStory("story0", apikey)
	assert.Nil(t, err)
	assert.Equal(t, "to be or not to be, that is the question?", content)

}
