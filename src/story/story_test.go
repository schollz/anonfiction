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
	err = Update("story0", apikey, "being", "to be or not to be, that is the question", []string{"questions"})
	assert.Nil(t, err)

	content, err := GetStory("story0", apikey)
	assert.Nil(t, err)
	assert.Equal(t, "to be or not to be, that is the question", content)

	err = Update("story0", apikey, "being", "to be or not to be, that is the question?", []string{"questions"})
	assert.Nil(t, err)

	content, err = GetStory("story0", apikey)
	assert.Nil(t, err)
	assert.Equal(t, "to be or not to be, that is the question?", content)

	err = Update("asdfasdf", "incorrectapikey", "being", "to be or not to be, that is the question", []string{"questions"})
	assert.NotNil(t, err)

	s := "<div>This is the first paragraph.<br><br>This has two lines.<br>But it should be one paragraph.<br><br>This&nbsp;<em>has italic</em>.&nbsp;<strong>This</strong> is bold. This is the third paragraph.</div>"
	ss := ConvertTrix(s)
	assert.Equal(t, "This is the first paragraph.", ss[0])
	assert.Equal(t, "This has two lines. But it should be one paragraph.", ss[1])
	assert.Equal(t, "This&nbsp;<em>has italic</em>.&nbsp;<strong>This</strong> is bold. This is the third paragraph.", ss[2])
}
