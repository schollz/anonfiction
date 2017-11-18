package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/schollz/storiesincognito/src/story"
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
		// 	c.Redirect(302, "/signin")
		// }
		storyID := c.DefaultQuery("story", utils.NewAPIKey())
		s, err := story.Get(storyID)
		if err != nil {
			c.HTML(http.StatusOK, "write.tmpl", MainView{
				StoryID:  storyID,
				APIKey:   GetSignedInUserAPIKey(c),
				SignedIn: true,
			})
		} else {
			log.Println(s.Content.GetCurrent())
			q := template.HTML(`<em>hi</em><b>test</b><br><br>paragraph><<>`)
			log.Println(q)
			c.HTML(http.StatusOK, "write.tmpl", MainView{
				StoryID:   storyID,
				APIKey:    GetSignedInUserAPIKey(c),
				SignedIn:  true,
				StoryHTML: q,
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
			c.Redirect(302, "/signin")
		}
		c.HTML(http.StatusOK, "profile.tmpl", MainView{
			SignedIn: true,
		})
	})
	router.GET("/read", func(c *gin.Context) {
		c.HTML(http.StatusOK, "read.tmpl", MainView{
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/topics", func(c *gin.Context) {
		c.HTML(http.StatusOK, "topics.tmpl", MainView{
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/signin", func(c *gin.Context) {
		if IsSignedIn(c) {
			c.Redirect(302, "/profile")
			return
		}
		c.HTML(http.StatusOK, "login.tmpl", MainView{
			SignedIn: false,
		})
	})
	router.GET("/signup", func(c *gin.Context) {
		if IsSignedIn(c) {
			c.Redirect(302, "/profile")
		}
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			SignedIn: false,
		})
	})
	router.GET("/signout", func(c *gin.Context) {
		SignOutUser(c)
		c.Redirect(302, "/")
	})
	router.GET("/admin", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.tmpl", MainView{
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/terms", func(c *gin.Context) {
		c.HTML(http.StatusOK, "terms.tmpl", MainView{
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy.tmpl", MainView{
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(302, "/static/img/meta/favicon.ico")
	})
	router.POST("/story", handlePOSTStory)
	router.POST("/signup", handlePOSTSignup)
	router.POST("/signin", handlePOSTSignin)
	router.Run(":" + port)
}

type MainView struct {
	Title        string
	ErrorMessage string
	InfoMessage  string
	Landing      bool
	SignedIn     bool
	// Story stuff
	Topic     string
	APIKey    string
	StoryID   string
	Keywords  string
	StoryHTML template.HTML
}

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "landing.tmpl", MainView{
		Landing: true,
	})
}

func handlePOSTStory(c *gin.Context) {
	type FormInput struct {
		Content  string `form:"content" json:"content" binding:"required"`
		Keywords string `form:"keywords" json:"keywords"`
		APIKey   string `form:"apikey" json:"apikey" binding:"required"`
		StoryID  string `form:"storyid" json:"storyid" binding:"required"`
		Topic    string `form:"storyid" json:"storyid" binding:"required"`
	}
	var form FormInput
	if err := c.ShouldBind(&form); err == nil {
		log.Println(form)
		form.Content = strings.Replace(form.Content, `"`, "&quot;", -1)
		keywords := strings.Split(form.Keywords, ",")
		err := story.Update(form.StoryID, form.APIKey, form.Topic, form.Content, keywords)
		var infoMessage, errorMessage string
		if err != nil {
			errorMessage = err.Error()
		} else {
			infoMessage = "Updated your story"
		}
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			StoryID:      form.StoryID,
			APIKey:       form.APIKey,
			Topic:        form.Topic,
			Keywords:     strings.Join(keywords, ", "),
			ErrorMessage: errorMessage,
			InfoMessage:  infoMessage,
			StoryHTML:    template.HTML(form.Content),
		})
	} else {
		c.JSON(200, gin.H{"error": err.Error()})
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
			c.Redirect(302, "/profile")
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
		c.Redirect(302, "/profile")
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
		return "uhoh"
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
