package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/schollz/storiesincognito/src/story"
	"github.com/schollz/storiesincognito/src/topic"
	"github.com/schollz/storiesincognito/src/user"
	"github.com/schollz/storiesincognito/src/utils"
)

type SessionKey struct {
	APIKey   string
	LastSeen time.Time
}

type Session struct {
	Keys map[string]SessionKey
	sync.RWMutex
}

var (
	port           string
	currentSession Session
)

const (
	TopicDB = "topics.db.json"
)

func init() {
	currentSession.Lock()
	currentSession.Keys = make(map[string]SessionKey)
	currentSession.Unlock()
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
	router.GET("/", handleIndex)
	router.GET("/write", func(c *gin.Context) {
		// if !IsSignedIn(c) {
		// 	SignInAndContinueOn(c)
		// 	return
		// }
		storyID := c.DefaultQuery("story", utils.NewAPIKey())
		topicName := c.DefaultQuery("topic", "")
		t, err := topic.Get(TopicDB, topicName)
		if err != nil {
			t, _ = topic.Default(TopicDB, false)
		}
		s, err := story.Get(storyID)
		if err != nil {
			c.HTML(http.StatusOK, "write.tmpl", MainView{
				StoryID:  storyID,
				APIKey:   GetSignedInUserAPIKey(c),
				SignedIn: true,
				Topic:    t,
				IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
			})
		} else {
			c.HTML(http.StatusOK, "write.tmpl", MainView{
				StoryID:  storyID,
				APIKey:   GetSignedInUserAPIKey(c),
				SignedIn: true,
				Story:    s,
				TrixAttr: template.HTMLAttr(`value="` + s.Content.GetCurrent() + `"`),
				Topic:    t,
				IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
			})
		}
	})
	router.GET("/upload", func(c *gin.Context) {
		if !IsSignedIn(c) {
			c.Redirect(302, "/signin")
		}
		c.HTML(http.StatusOK, "upload.tmpl", MainView{
			SignedIn: true,
		})
	})
	router.GET("/profile", func(c *gin.Context) {
		if !IsSignedIn(c) {
			SignInAndContinueOn(c)
			return
		}
		stories, _ := story.ListByUser(GetSignedInUserAPIKey(c))
		c.HTML(http.StatusOK, "profile.tmpl", MainView{
			SignedIn: true,
			Stories:  stories,
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/read", func(c *gin.Context) {
		var stories []story.Story
		var s story.Story
		var t topic.Topic
		var err error
		var nextStory, previousStory string
		storyID := c.DefaultQuery("story", "")
		topicName := c.DefaultQuery("topic", "")
		if storyID != "" {
			s, err = story.Get(storyID)
			if err != nil {
				c.HTML(http.StatusOK, "error.tmpl", MainView{
					ErrorMessage: err.Error(),
					ErrorCode:    "503",
				})
				return
			}
			topicName = s.Topic
		}

		t, err = topic.Get(TopicDB, topicName)
		if err != nil {
			t, _ = topic.Default(TopicDB, true)
		}
		stories, err = story.ListByTopic(t.Name)
		if err != nil {
			stories = []story.Story{s}
		}
		storyI := 0
		if storyID != "" {
			for i := range stories {
				storyI = i
				if stories[i].ID == storyID {
					break
				}
			}
		} else {
			s = stories[storyI]
		}
		if storyI > 0 {
			previousStory = stories[storyI-1].ID
		}
		if storyI < len(stories)-1 {
			nextStory = stories[storyI+1].ID
		}
		log.Println(s)

		c.HTML(http.StatusOK, "read.tmpl", MainView{
			SignedIn: IsSignedIn(c),
			Topic:    t,
			Story:    s,
			Next:     nextStory,
			Previous: previousStory,
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/topics", func(c *gin.Context) {
		topics, err := topic.Load(TopicDB)
		if err != nil {
			c.HTML(http.StatusOK, "error.tmpl", MainView{
				ErrorMessage: err.Error(),
				ErrorCode:    "503",
				IsAdmin:      user.IsAdmin(GetSignedInUserAPIKey(c)),
			})
			return
		}
		c.HTML(http.StatusOK, "topics.tmpl", MainView{
			Topics:   topics,
			SignedIn: IsSignedIn(c),
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/signin", func(c *gin.Context) {
		if IsSignedIn(c) {
			c.Redirect(302, "/profile")
			return
		}
		c.HTML(http.StatusOK, "login.tmpl", MainView{
			SignedIn: false,
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusOK, "error.tmpl", MainView{
			ErrorCode:    "404",
			ErrorMessage: "Sorry, we can't find the page you are looking for.",
			SignedIn:     false,
			IsAdmin:      user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/signup", func(c *gin.Context) {
		if IsSignedIn(c) {
			c.Redirect(302, "/profile")
		}
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			SignedIn: false,
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/signout", func(c *gin.Context) {
		SignOutUser(c)
		c.Redirect(302, "/")
	})
	router.GET("/admin", func(c *gin.Context) {
		if !IsSignedIn(c) {
			SignInAndContinueOn(c)
			return
		}
		u, err := user.GetByAPIKey(GetSignedInUserAPIKey(c))
		if err != nil {
			c.HTML(http.StatusOK, "error.tmpl", MainView{
				ErrorCode:    "503",
				ErrorMessage: err.Error(),
				SignedIn:     true,
				IsAdmin:      user.IsAdmin(GetSignedInUserAPIKey(c)),
			})
		}
		if !u.IsAdmin {
			c.HTML(http.StatusOK, "error.tmpl", MainView{
				ErrorCode:    "401",
				ErrorMessage: "Unauthorized",
				SignedIn:     true,
				IsAdmin:      user.IsAdmin(GetSignedInUserAPIKey(c)),
			})
		}
		stories, err := story.All()
		if err != nil {
			c.HTML(http.StatusOK, "error.tmpl", MainView{
				ErrorCode:    "503",
				ErrorMessage: err.Error(),
				SignedIn:     true,
				IsAdmin:      user.IsAdmin(GetSignedInUserAPIKey(c)),
			})
		}
		c.HTML(http.StatusOK, "admin.tmpl", MainView{
			SignedIn: IsSignedIn(c),
			Stories:  stories,
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/terms", func(c *gin.Context) {
		c.HTML(http.StatusOK, "terms.tmpl", MainView{
			SignedIn: IsSignedIn(c),
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy.tmpl", MainView{
			SignedIn: IsSignedIn(c),
			IsAdmin:  user.IsAdmin(GetSignedInUserAPIKey(c)),
		})
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(302, "/static/img/meta/favicon.ico")
	})
	router.POST("/write", handlePOSTStory)
	router.POST("/signup", handlePOSTSignup)
	router.POST("/signin", handlePOSTSignin)
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
	Next         string
	Previous     string
	TrixAttr     template.HTMLAttr
}

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "landing.tmpl", MainView{
		Landing: true,
		IsAdmin: user.IsAdmin(GetSignedInUserAPIKey(c)),
	})
}

func handlePOSTStory(c *gin.Context) {
	type FormInput struct {
		Content   string `form:"content" json:"content" binding:"required"`
		Keywords  string `form:"keywords" json:"keywords"`
		APIKey    string `form:"apikey" json:"apikey" binding:"required"`
		StoryID   string `form:"storyid" json:"storyid" binding:"required"`
		Topic     string `form:"topic" json:"topic" binding:"required"`
		Published string `form:"published" json:"published"`
	}
	defaultTopic, _ := topic.Default(TopicDB, false)
	var form FormInput
	if err := c.ShouldBind(&form); err == nil {
		log.Println(form)
		// check topic is valid
		t, err := topic.Get(TopicDB, form.Topic)
		if err != nil {
			c.HTML(http.StatusOK, "error.tmpl", MainView{
				ErrorCode:    "503",
				ErrorMessage: err.Error(),
				IsAdmin:      user.IsAdmin(GetSignedInUserAPIKey(c)),
			})
			return
		}
		form.Content = strings.Replace(form.Content, `"`, "&quot;", -1)
		keywords := strings.Split(form.Keywords, ",")
		s, err := story.Update(form.StoryID, form.APIKey, form.Topic, form.Content, keywords, form.Published == "on")
		log.Println(form.Published, s.Published)
		fmt.Println(err)
		var infoMessage, errorMessage string
		if err != nil {
			err = errors.Wrap(err, "story not submitted")
			errorMessage = err.Error()
		} else {
			infoMessage = "Updated your story"
		}
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			StoryID:      form.StoryID,
			APIKey:       form.APIKey,
			Topic:        t,
			ErrorMessage: errorMessage,
			InfoMessage:  infoMessage,
			Story:        s,
			IsAdmin:      user.IsAdmin(form.APIKey),
			TrixAttr:     template.HTMLAttr(`value="` + s.Content.GetCurrent() + `"`),
			SignedIn:     true,
		})
	} else {
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			StoryID:      form.StoryID,
			APIKey:       form.APIKey,
			ErrorMessage: err.Error(),
			Topic:        defaultTopic,
			IsAdmin:      user.IsAdmin(form.APIKey),
			SignedIn:     true,
		})
	}
}

func handlePOSTSignup(c *gin.Context) {
	type FormInput struct {
		Email    string `form:"email" json:"email" binding:"required"`
		Password string `form:"password" json:"password"`
		Language string `form:"language" json:"language"`
		Digest   string `form:"digest" json:"digest"`
	}
	var form FormInput
	if err := c.ShouldBind(&form); err == nil {
		log.Println(form)
		form.Email = strings.ToLower(form.Email)
		log.Println(user.UserExists(form.Email))
		if user.UserExists(form.Email) {
			c.HTML(http.StatusOK, "signup.tmpl", MainView{
				ErrorMessage: "Email already exists",
			})
			return
		}
		form.Password = strings.TrimSpace(form.Password)
		if len(form.Password) == 0 {
			c.HTML(http.StatusOK, "signup.tmpl", MainView{
				ErrorMessage: "Must choose better password",
			})
			return
		}
		log.Println("Adding new user " + form.Email)
		err := user.Add(form.Email, form.Password, form.Language, form.Digest == "on")
		log.Println(err)
		if err != nil {
			c.HTML(http.StatusOK, "signup.tmpl", MainView{
				ErrorMessage: "Email already exists",
			})
		} else {
			log.Println("redirecting to profile")
			u, err := user.Get(form.Email)
			if err != nil {
				c.HTML(http.StatusOK, "signup.tmpl", MainView{
					ErrorMessage: err.Error(),
				})
				return
			}
			SignInUser(u.APIKey, c)
		}
	} else {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			ErrorMessage: err.Error(),
		})
	}
}

func handlePOSTSignin(c *gin.Context) {
	type FormInput struct {
		Email    string `form:"email" json:"email" binding:"required"`
		Password string `form:"password" json:"password"`
	}
	var form FormInput
	if err := c.ShouldBind(&form); err == nil {
		form.Email = strings.ToLower(form.Email)
		if !user.UserExists(form.Email) {
			c.HTML(http.StatusOK, "login.tmpl", MainView{
				ErrorMessage: "User does not exist",
			})
			return
		}
		form.Password = strings.TrimSpace(form.Password)
		apikey, err := user.Validate(form.Email, form.Password)
		if err != nil {
			c.HTML(http.StatusOK, "login.tmpl", MainView{
				ErrorMessage: err.Error(),
			})
			return
		}
		SignInUser(apikey, c)
	} else {
		c.HTML(http.StatusOK, "login.tmpl", MainView{
			ErrorMessage: err.Error(),
		})
	}
}

func IsSignedIn(c *gin.Context) bool {
	cookies := sessions.Default(c)
	clientKey := cookies.Get("sessionkey")
	if clientKey == nil {
		return false
	}
	currentSession.Lock()
	defer currentSession.Unlock()
	_, ok := currentSession.Keys[clientKey.(string)]
	if ok {
		currentSession.Keys[utils.NewAPIKey()] = SessionKey{
			APIKey:   currentSession.Keys[clientKey.(string)].APIKey,
			LastSeen: time.Now(),
		}
	}
	return ok
}

func GetSignedInUserAPIKey(c *gin.Context) string {
	if !IsSignedIn(c) {
		return ""
	}
	cookies := sessions.Default(c)
	clientKey := cookies.Get("sessionkey")
	if clientKey == nil {
		return ""
	}
	currentSession.Lock()
	defer currentSession.Unlock()
	key := currentSession.Keys[clientKey.(string)].APIKey
	return key
}

func SignInUser(apikey string, c *gin.Context) {
	currentSession.Lock()
	defer currentSession.Unlock()
	tempAPIKey := utils.NewAPIKey()
	currentSession.Keys[tempAPIKey] = SessionKey{
		APIKey:   apikey,
		LastSeen: time.Now(),
	}
	cookies := sessions.Default(c)
	cookies.Set("sessionkey", tempAPIKey)
	err := cookies.Save()
	if err != nil {
		log.Println(err)
	}
	continueOn := cookies.Get("continueon")
	if continueOn != nil {
		c.Redirect(302, continueOn.(string))
	} else {
		c.Redirect(302, "/profile")
	}
}

func SignOutUser(c *gin.Context) {
	cookies := sessions.Default(c)
	clientKey := cookies.Get("sessionkey")
	if clientKey == nil {
		return
	}
	currentSession.Lock()
	defer currentSession.Unlock()
	_, ok := currentSession.Keys[clientKey.(string)]
	if ok {
		delete(currentSession.Keys, clientKey.(string))
	}
	cookies.Clear()
}

func SignInAndContinueOn(c *gin.Context) {
	fmt.Println(c.Request.URL.String())
	cookies := sessions.Default(c)
	cookies.Set("continueon", c.Request.URL.String())
	err := cookies.Save()
	if err != nil {
		log.Println(err)
	}
	c.Redirect(302, "/signin")
}
