package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/schollz/jsonstore"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/schollz/storiesincognito/src/encrypt"
	"github.com/schollz/storiesincognito/src/story"
	"github.com/schollz/storiesincognito/src/topic"
	"github.com/schollz/storiesincognito/src/user"
)

var (
	port string
	keys *jsonstore.JSONStore
)

const (
	TopicDB = "topics.db.json"
)

func init() {
	var err error
	keys, err = jsonstore.Open("keys.json")
	if err != nil {
		keys = new(jsonstore.JSONStore)
	}
}

func main() {
	flag.StringVar(&port, "port", "3001", "port of server")
	flag.Parse()
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	store := sessions.NewCookieStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "landing.tmpl", MainView{
			Landing:  true,
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/write", func(c *gin.Context) {
		storyID := c.DefaultQuery("story", xid.New().String())
		topicName := c.DefaultQuery("topic", "")
		t, err := topic.Get(TopicDB, topicName)
		if err != nil {
			t, _ = topic.Default(TopicDB, false)
		}
		userID, err := GetUserIDFromCookie(c)
		if err != nil {
			userID = user.AnonymousUserID()
		}
		fmt.Println(storyID)
		s, err := story.Get(storyID)
		fmt.Println(s)
		if err != nil {
			log.Println(err)
			s = story.New(userID, t.Name, "", "", []string{})
		}
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Story:    s,
			TrixAttr: template.HTMLAttr(`value="` + s.Content.GetCurrent() + `"`),
		})
	})
	router.GET("/upload", func(c *gin.Context) {
		if !IsSignedIn(c) {
			c.Redirect(302, "/login")
		}
		c.HTML(http.StatusOK, "upload.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/profile", func(c *gin.Context) {
		if !IsSignedIn(c) {
			SignInAndContinueOn(c)
			return
		}
		userID, err := GetUserIDFromCookie(c)
		if err != nil {
			ShowError(err, c)
			return
		}
		stories, _ := story.ListByUser(userID)
		c.HTML(http.StatusOK, "profile.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Stories:  stories,
		})
	})
	router.GET("/delete", func(c *gin.Context) {
		if !IsSignedIn(c) {
			SignInAndContinueOn(c)
			return
		}
		storyID := c.DefaultQuery("story", "")
		s, err := story.Get(storyID)
		if err != nil {
			ShowError(err, c)
			return
		}
		err = s.Delete()
		if err != nil {
			ShowError(err, c)
			return
		}
		userID, err := GetUserIDFromCookie(c)
		if err != nil {
			ShowError(err, c)
			return
		}
		stories, _ := story.ListByUser(userID)
		c.HTML(http.StatusOK, "profile.tmpl", MainView{
			IsAdmin:     IsAdmin(c),
			SignedIn:    IsSignedIn(c),
			Stories:     stories,
			InfoMessage: "Story '" + storyID + "' deleted",
		})
	})
	router.GET("/read", func(c *gin.Context) {
		var err error
		var s story.Story
		var t topic.Topic
		var nextStory, previousStory string
		storyID := c.DefaultQuery("story", "")
		topicName := c.DefaultQuery("topic", "")
		if storyID != "" {
			s, err = story.Get(storyID)
			if err != nil {
				ShowError(err, c)
				return
			}
			topicName = s.Topic
		}
		t, _ = topic.Get(TopicDB, topicName)
		c.HTML(http.StatusOK, "read.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Topic:    t,
			Story:    s,
			Next:     nextStory,
			Previous: previousStory,
		})
	})
	router.GET("/topics", func(c *gin.Context) {
		topics, err := topic.Load(TopicDB)
		if err != nil {
			ShowError(err, c)
			return
		}
		c.HTML(http.StatusOK, "topics.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Topics:   topics,
		})
	})
	router.GET("/login", func(c *gin.Context) {
		if IsSignedIn(c) {
			c.Redirect(302, "/profile")
			return
		}
		uuid := c.DefaultQuery("key", "")
		if uuid == "" {
			c.HTML(http.StatusOK, "login.tmpl", MainView{
				IsAdmin:  IsAdmin(c),
				SignedIn: IsSignedIn(c),
			})
			return
		}
		err := SignIn(uuid, c)
		if err != nil {
			c.HTML(http.StatusOK, "login.tmpl", MainView{
				ErrorMessage: err.Error(),
				IsAdmin:      IsAdmin(c),
				SignedIn:     IsSignedIn(c),
			})
			return
		}
		c.Redirect(302, "/profile")
	})
	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusOK, "error.tmpl", MainView{
			IsAdmin:      IsAdmin(c),
			SignedIn:     IsSignedIn(c),
			ErrorCode:    "404",
			ErrorMessage: "Sorry, we can't find the page you are looking for.",
		})
	})
	router.GET("/signup", func(c *gin.Context) {
		if IsSignedIn(c) {
			c.Redirect(302, "/profile")
		}
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/signout", func(c *gin.Context) {
		SignOut(c)
		c.Redirect(302, "/")
	})
	router.GET("/admin", func(c *gin.Context) {
		if !IsSignedIn(c) {
			SignInAndContinueOn(c)
			return
		}

		if !IsAdmin(c) {
			ShowError(errors.New("Not admin"), c)
			return
		}
		stories, err := story.All()
		if err != nil {
			ShowError(err, c)
			return
		}
		users, err := user.All()
		if err != nil {
			ShowError(err, c)
			return
		}
		c.HTML(http.StatusOK, "admin.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Stories:  stories,
			Users:    users,
		})
	})
	router.GET("/terms", func(c *gin.Context) {
		c.HTML(http.StatusOK, "terms.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/about", func(c *gin.Context) {
		c.HTML(http.StatusOK, "about.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(302, "/static/img/meta/favicon.ico")
	})
	router.POST("/write", handlePOSTStory)
	router.POST("/login", handlePOSTSignup)
	router.Run(":" + port)
}

type MainView struct {
	IsAdmin      bool
	Title        string
	ErrorMessage string
	ErrorCode    string
	InfoMessage  string
	Landing      bool
	SignedIn     bool
	Story        story.Story
	Topic        topic.Topic
	APIKey       string
	StoryID      string
	Topics       []topic.Topic
	Stories      []story.Story
	Users        []user.User
	Next         string
	Previous     string
	TrixAttr     template.HTMLAttr
}

func handlePOSTStory(c *gin.Context) {
	type FormInput struct {
		StoryID     string `form:"storyid" json:"storyid"`
		Topic       string `form:"topic" json:"topic" binding:"required"`
		Content     string `form:"content" json:"content" binding:"required"`
		Description string `form:"description" json:"description"`
		Keywords    string `form:"keywords" json:"keywords"`
		Published   string `form:"published" json:"published"`
	}
	defaultTopic, _ := topic.Default(TopicDB, false)
	var form FormInput
	if err := c.ShouldBind(&form); err == nil {
		log.Println(form)
		form.Content = strings.Replace(form.Content, `"`, "&quot;", -1)
		keywords := strings.Split(form.Keywords, ",")
		var s story.Story
		userID, err := GetUserIDFromCookie(c)
		if err != nil {
			userID = user.AnonymousUserID()
		}
		s, err = story.Get(form.StoryID)
		isNewStory := false
		if err != nil {
			s = story.New(userID, form.Topic, "", "", []string{})
			s.ID = form.StoryID
			isNewStory = true
		}
		s.Content.Update(form.Content)
		s.Keywords = keywords
		s.Description = form.Description
		s.Published = form.Published == "on"
		if !isNewStory && userID == user.AnonymousUserID() {
			err = errors.New("cannot update an anonymous story")
		} else if userID != s.UserID {
			err = errors.New("cannot update someone elses story")
		} else {
			err = s.Save()
		}
		var infoMessage, errorMessage string
		if err != nil {
			err = errors.Wrap(err, "story not submitted")
			errorMessage = err.Error()
		} else {
			infoMessage = "updated story"
		}
		fmt.Println("storyID", s.ID)
		fmt.Println("userID", s.UserID)
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			IsAdmin:      IsAdmin(c),
			SignedIn:     IsSignedIn(c),
			InfoMessage:  infoMessage,
			ErrorMessage: errorMessage,
			Story:        s,
			TrixAttr:     template.HTMLAttr(`value="` + s.Content.GetCurrent() + `"`),
		})
	} else {
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			IsAdmin:      IsAdmin(c),
			SignedIn:     IsSignedIn(c),
			ErrorMessage: err.Error(),
			Topic:        defaultTopic,
		})
	}
}

func handlePOSTSignup(c *gin.Context) {
	defer jsonstore.Save(keys, "keys.json")
	type FormInput struct {
		Email    string `form:"email" json:"email" binding:"required"`
		Language string `form:"language" json:"language"`
		Digest   string `form:"digest" json:"digest"`
	}
	var form FormInput
	if err := c.ShouldBind(&form); err == nil {
		form.Email = strings.ToLower(form.Email)
		userID, err := user.GetID(form.Email)
		if err != nil {
			// create user
			err = user.Add(form.Email, form.Language, form.Digest == "on")
			if err != nil {
				ShowError(err, c)
				return
			}
			userID, err = user.GetID(form.Email)
			if err != nil {
				log.Fatal(err)
			}
		}

		// add to validation keys
		uuid := xid.New().String()
		err = keys.Set("uuid:"+uuid, userID)
		if err != nil {
			log.Fatal(err)
		}
		go jsonstore.Save(keys, "keys.json")
		// send the link to email
		fmt.Println("http://localhost:" + port + "/login?key=" + uuid)
		c.HTML(http.StatusOK, "login.tmpl", MainView{
			InfoMessage: "http://localhost:" + port + "/login?key=" + uuid,
			IsAdmin:     IsAdmin(c),
			SignedIn:    IsSignedIn(c),
		})
	} else {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			ErrorMessage: err.Error(),
		})
	}
}

func getCookie(key string, c *gin.Context) (cookie string, err error) {
	cookies := sessions.Default(c)
	data := cookies.Get(key)
	if data == nil {
		err = errors.New("Cookie not available for '" + key + "'")
		return
	}
	cookie, err = encrypt.Decrypt(data.(string), "secrete")
	return
}

func setCookie(key, value string, c *gin.Context) (err error) {
	cookies := sessions.Default(c)
	encrypted, err := encrypt.Encrypt(value, "secrete")
	if err != nil {
		return
	}
	cookies.Set(key, encrypted)
	err = cookies.Save()
	return
}

func IsSignedIn(c *gin.Context) bool {
	apikey, err := getCookie("apikey", c)
	if err != nil {
		return false
	}
	var userID string
	err = keys.Get("apikey:"+apikey, &userID)
	if err == nil {
		return true
	}
	return false
}

func IsAdmin(c *gin.Context) bool {
	apikey, err := getCookie("apikey", c)
	if err != nil {
		return false
	}
	var userID string
	err = keys.Get("apikey:"+apikey, &userID)
	if err != nil {
		return false
	}
	var foo string
	err = keys.Get("admin:"+userID, &foo)
	return err == nil
}

func SignIn(uuid string, c *gin.Context) (err error) {
	defer jsonstore.Save(keys, "keys.json")
	var userID string
	// First check to see if its in the validator
	err = keys.Get("uuid:"+uuid, &userID)
	if err != nil {
		err = errors.New("Must request new sign-in")
		return
	}

	// Generate a new API key
	apikey := xid.New().String()
	err = keys.Set("apikey:"+apikey, userID)
	if err != nil {
		return
	}

	// Set the cookie with the API key
	err = setCookie("apikey", apikey, c)
	if err != nil {
		log.Println(err)
	}

	// Delete the UUID to prevent being used again
	keys.Delete("uuid:" + uuid)

	// Check the continue on if it needs to be done
	cookies := sessions.Default(c)
	continueOn := cookies.Get("continueon")
	if continueOn != nil {
		c.Redirect(302, continueOn.(string))
	} else {
		c.Redirect(302, "/profile")
	}
	return nil
}

func GetUserIDFromCookie(c *gin.Context) (userID string, err error) {
	apikey, err := getCookie("apikey", c)
	if err != nil {
		return
	}
	err = keys.Get("apikey:"+apikey, &userID)
	return
}

func SignOut(c *gin.Context) (err error) {
	defer jsonstore.Save(keys, "keys.json")
	cookies := sessions.Default(c)
	apikey, err := getCookie("apikey", c)
	if err != nil {
		return
	}
	keys.Delete("apikey:" + apikey)
	cookies.Clear()
	return
}

func SignInAndContinueOn(c *gin.Context) {
	fmt.Println(c.Request.URL.String())
	cookies := sessions.Default(c)
	cookies.Set("continueon", c.Request.URL.String())
	err := cookies.Save()
	if err != nil {
		log.Println(err)
	}
	c.Redirect(302, "/login")
}

func ShowError(err error, c *gin.Context) {
	c.HTML(http.StatusOK, "error.tmpl", MainView{
		IsAdmin:      IsAdmin(c),
		SignedIn:     IsSignedIn(c),
		ErrorMessage: err.Error(),
		ErrorCode:    "503",
	})
}
