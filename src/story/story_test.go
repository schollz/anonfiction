package story

import (
	"html/template"
	"os"
	"testing"

	"github.com/schollz/anonfiction/src/user"
	"github.com/stretchr/testify/assert"
)

func TestStory(t *testing.T) {
	os.Remove("user_test.db")
	os.Remove("story_test.db")
	user.DB = "user_test.db"
	DB = "story_test.db"
	err := user.Add("zack",  "english", false)
	assert.Nil(t, err)


	s := "<div>This is the first paragraph.<br><br>This has two lines.<br>But it should be one paragraph.<br><br>This&nbsp;<em>has italic</em>.&nbsp;<strong>This</strong> is bold. This is the third paragraph.</div>"
	ss := ConvertTrix(s)
	assert.Equal(t, template.HTML("<span class=\"startsentence\">This is the first paragraph."), ss[0])
	assert.Equal(t, template.HTML("This has two lines. But it should be one paragraph."), ss[1])
	assert.Equal(t, template.HTML("This&nbsp;<em>has italic</em>.&nbsp;<strong>This</strong> is bold. This is the third paragraph."), ss[2])
}
