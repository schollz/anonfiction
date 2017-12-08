package story

import (
	"html/template"
	"os"
	"testing"
	"time"

	"github.com/schollz/anonfiction/src/user"
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
	_, err = Update("story0", apikey, "being", "to be or not to be, that is the question", []string{"questions"}, false)
	assert.Nil(t, err)

	content, err := GetStory("story0", apikey)
	assert.Nil(t, err)
	assert.Equal(t, "to be or not to be, that is the question", content)

	_, err = Update("story0", apikey, "being", "to be or not to be, that is the question?", []string{"questions"}, false)
	assert.Nil(t, err)

	content, err = GetStory("story0", apikey)
	assert.Nil(t, err)
	assert.Equal(t, "to be or not to be, that is the question?", content)

	_, err = Update("asdfasdf", "incorrectapikey", "being", "to be or not to be, that is the question", []string{"questions"}, false)
	assert.NotNil(t, err)

	time.Sleep(1)
	Update("story1", apikey, "being", "a new story", []string{"questions"}, false)
	user.Add("zack2", "123", "english", false)
	apikey2, _ := user.Validate("zack2", "123")
	Update("anotherstory", apikey2, "being", "a new story", []string{"questions"}, false)
	stories, err := ListByUser(apikey)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(stories))
	assert.True(t, stories[0].Date.After(stories[1].Date))
	stories, err = ListByUser(apikey2)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(stories))
	stories, err = ListByTopic("being")
	assert.Nil(t, err)
	assert.Equal(t, 3, len(stories))

	s := "<div>This is the first paragraph.<br><br>This has two lines.<br>But it should be one paragraph.<br><br>This&nbsp;<em>has italic</em>.&nbsp;<strong>This</strong> is bold. This is the third paragraph.</div>"
	ss := ConvertTrix(s)
	assert.Equal(t, template.HTML("<span class=\"leader f4\">This is the first paragraph."), ss[0])
	assert.Equal(t, template.HTML("This has two lines. But it should be one paragraph."), ss[1])
	assert.Equal(t, template.HTML("This&nbsp;<em>has italic</em>.&nbsp;<strong>This</strong> is bold. This is the third paragraph."), ss[2])
}
