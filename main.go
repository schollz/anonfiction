package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/schollz/storiesincognito/src/story"
	"github.com/schollz/storiesincognito/src/user"
	"github.com/schollz/storiesincognito/src/utils"
)

var (
	port string
)

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
		storyID := c.DefaultQuery("story", utils.NewAPIKey())
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			StoryID: storyID,
			APIKey:  "foo",
		})
	})
	router.GET("/upload", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.tmpl", MainView{})
	})
	router.GET("/profile", func(c *gin.Context) {
		c.HTML(http.StatusOK, "profile.tmpl", MainView{})
	})
	router.GET("/read", func(c *gin.Context) {
		c.HTML(http.StatusOK, "read.tmpl", MainView{})
	})
	router.GET("/topics", func(c *gin.Context) {
		c.HTML(http.StatusOK, "topics.tmpl", MainView{})
	})
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.tmpl", MainView{})
	})
	router.GET("/signup", func(c *gin.Context) {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{})
	})
	router.GET("/admin", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.tmpl", MainView{})
	})
	router.GET("/terms", func(c *gin.Context) {
		c.HTML(http.StatusOK, "terms.tmpl", MainView{})
	})
	router.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy.tmpl", MainView{})
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(302, "/static/img/meta/favicon.ico")
	})
	router.POST("/story", handlePOSTStory)
	router.POST("/signup", handlePOSTSignup)
	router.Run(":" + port)
}

type MainView struct {
	Title        string
	ErrorMessage string
	InfoMessage  string
	Landing      bool
	// Story stuff
	Topic    string
	APIKey   string
	StoryID  string
	Keywords string
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
		})
	} else {
		c.JSON(200, gin.H{"error": err.Error()})
	}
}

func handlePOSTSignup(c *gin.Context) {
	type FormInput struct {
		Username string `form:"username" json:"username" binding:"required"`
		Password string `form:"password" json:"password"`
		Language string `form:"language" json:"language"`
		Digest   string `form:"digest" json:"digest"`
	}
	var form FormInput
	if err := c.ShouldBind(&form); err == nil {
		log.Println(form)
		log.Println(user.UserExists(form.Username))
		if user.UserExists(form.Username) {
			c.HTML(http.StatusOK, "signup.tmpl", MainView{
				ErrorMessage: "Username already exists",
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
		log.Println("Adding new user " + form.Username)
		err := user.Add(form.Username, form.Password, form.Language, form.Digest == "on")
		log.Println(err)
		if err != nil {
			c.HTML(http.StatusOK, "signup.tmpl", MainView{
				ErrorMessage: "Username already exists",
			})
		} else {
			log.Println("redirecting to profile")
			c.Redirect(302, "/profile")
		}
	} else {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			ErrorMessage: err.Error(),
		})
	}
}
