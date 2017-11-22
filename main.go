package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/matcornic/hermes"
	"github.com/pkg/errors"
	"github.com/schollz/jsonstore"
	"github.com/schollz/storiesincognito/src/encrypt"
	"github.com/schollz/storiesincognito/src/story"
	"github.com/schollz/storiesincognito/src/topic"
	"github.com/schollz/storiesincognito/src/user"
	"github.com/schollz/storiesincognito/src/utils"
	"github.com/sirupsen/logrus"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"
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
		logrus.WithFields(logrus.Fields{
			"func": "init",
		}).Error(err.Error())
		keys = new(jsonstore.JSONStore)
	}
}

func slugify(s string) string {
	return strings.ToLower(strings.Join(strings.Split(strings.TrimSpace(s), " "), "-"))
}

func unslugify(s string) string {
	return strings.TrimSpace(strings.Title(strings.Join(strings.Split(s, "-"), " ")))
}

func main() {
	flag.StringVar(&port, "port", "3001", "port of server")
	flag.Parse()
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	store := sessions.NewCookieStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))
	router.SetFuncMap(template.FuncMap{
		"slugify":   slugify,
		"unslugify": unslugify,
	})
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "landing.tmpl", MainView{
			Landing:  true,
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/read/*actions", func(c *gin.Context) {
		i := c.DefaultQuery("i", "")
		actions := strings.Split(c.Param("actions"), "/")
		if len(actions) == 1 {
			c.Redirect(302, "/read/topic")
			return
		}
		action := actions[1]
		var id string
		if len(actions) > 2 {
			id = strings.TrimSpace(actions[2])
		}

		var err error
		var s story.Story
		var t topic.Topic
		var stories []story.Story
		var iNum int
		var nextStory, previousStory string
		if action == "story" {
			stories = make([]story.Story, 1)
			stories[0], err = story.Get(id)
			if err != nil {
				ShowError(err, c)
				return
			}
		} else if action == "keyword" {
			if id == "" {
				c.Redirect(302, "/read/topic/")
				return
			}
			stories, err = story.ListByKeyword(id)
		} else {
			if id == "" {
				t, _ := topic.Default(TopicDB, true)
				c.Redirect(302, "/read/topic/"+slugify(t.Name))
				return
			}
			stories, err = story.ListByTopic(unslugify(id))
			logrus.WithFields(logrus.Fields{
				"func": "handleRead",
			}).Infof("Found %d stories for '%s'", len(stories), unslugify(id))
		}
		if err != nil || len(stories) == 0 {
			c.HTML(http.StatusOK, "error.tmpl", MainView{
				IsAdmin:         IsAdmin(c),
				SignedIn:        IsSignedIn(c),
				InfoMessageHTML: template.HTML("No stories yet, <a href='/write'>why don't you write one?</a>"),
				ErrorCode:       "Uh oh!",
			})
			return
		}
		if i == "" {
			iNum = 1
		} else {
			iNum, err = strconv.Atoi(i)
			if err != nil || iNum > len(stories) {
				iNum = len(stories)
				c.Redirect(302, "/read/topic/"+id+"/?i="+strconv.Itoa(len(stories)))
				return
			}
			if iNum < 1 {
				c.Redirect(302, "/read/topic/"+id+"/?i="+strconv.Itoa(1))
				return
			}
		}
		s = stories[iNum-1]
		if iNum < len(stories) {
			nextStory = strconv.Itoa(iNum + 1)
		}
		if iNum > 1 {
			previousStory = strconv.Itoa(iNum - 1)
		}
		t, _ = topic.Get(TopicDB, s.Topic)
		c.HTML(http.StatusOK, "read.tmpl", MainView{
			IsAdmin:    IsAdmin(c),
			SignedIn:   IsSignedIn(c),
			Topic:      t,
			Story:      s,
			Next:       nextStory,
			Previous:   previousStory,
			NumStory:   iNum,
			NumStories: len(stories),
			Route:      action + "/" + id,
		})
	})
	router.GET("/write/*storyID", func(c *gin.Context) {
		storyID := c.Param("storyID")[1:]
		if len(storyID) == 0 {
			storyID = utils.NewAPIKey()
		}
		topics, err := topic.Active(TopicDB)
		if err != nil {
			ShowError(err, c)
			return
		}
		userID, err := GetUserIDFromCookie(c)
		if err != nil {
			userID = user.AnonymousUserID()
		}
		s, err := story.Get(storyID)
		if err != nil {
			s = story.New(userID, "", "", "", []string{})
		}
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Story:    s,
			Topics:   topics,
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
	router.POST("/write/*foo", handlePOSTStory)
	router.POST("/login", handlePOSTSignup)
	fmt.Println("Running at http://localhost:3001")
	router.Run(":" + port)
}

type MainView struct {
	IsAdmin         bool
	Title           string
	ErrorMessage    string
	ErrorCode       string
	InfoMessage     string
	InfoMessageHTML template.HTML
	Landing         bool
	SignedIn        bool
	Story           story.Story
	Topic           topic.Topic
	APIKey          string
	StoryID         string
	Topics          []topic.Topic
	Stories         []story.Story
	NumStory        int
	NumStories      int
	Users           []user.User
	Next            string
	Previous        string
	TrixAttr        template.HTMLAttr
	Route           string
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
	var form FormInput
	topics, _ := topic.Load(TopicDB)
	if err := c.ShouldBind(&form); err == nil {
		form.Content = strings.Replace(form.Content, `"`, "&quot;", -1)
		keywords := strings.Split(form.Keywords, ",")
		for i, keyword := range keywords {
			keywords[i] = slugify(keyword)
		}
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
		s.Topic = form.Topic
		s.Keywords = keywords
		s.Description = form.Description
		s.Published = form.Published == "on"
		if IsAdmin(c) {
			err = s.Save()
		} else if !isNewStory && userID == user.AnonymousUserID() {
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
			infoMessage = fmt.Sprintf("Story updated, read it at <a href='/read/story/%s'>/read/story/%s</a>", s.ID, s.ID)
		}
		fmt.Println("storyID", s.ID)
		fmt.Println("userID", s.UserID)
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
			InfoMessageHTML: template.HTML(infoMessage),
			ErrorMessage:    errorMessage,
			Story:           s,
			TrixAttr:        template.HTMLAttr(`value="` + s.Content.GetCurrent() + `"`),
			Topics:          topics,
		})
	} else {
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			IsAdmin:      IsAdmin(c),
			SignedIn:     IsSignedIn(c),
			ErrorMessage: err.Error(),
			Topics:       topics,
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
		uuid := utils.NewAPIKey()
		err = keys.Set("uuid:"+uuid, userID)
		if err != nil {
			log.Fatal(err)
		}
		go jsonstore.Save(keys, "keys.json")
		// send the link to email
		logrus.WithFields(logrus.Fields{
			"func": "handlePOSTSignup",
		}).Infof("http://localhost:%s/login?key=%s", port, uuid)
		sendEmail(form.Email, "Welcome!\n\nPlease use the following link to finish logging in:\n\nhttps://storiesincognito.org/login?key="+uuid+"\n\nNote: This link will only work once. Feel free to request new ones though!\n\nThanks!\n\n- Stories Incognito Team")
		c.HTML(http.StatusOK, "login.tmpl", MainView{
			InfoMessage: "You have been sent an email. Click the link in the email to login.",
			IsAdmin:     IsAdmin(c),
			SignedIn:    IsSignedIn(c),
		})
	} else {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			ErrorMessage: err.Error(),
		})
	}
}

func sendEmail(address, messageText string) {
	// Configure hermes by setting a theme and your product info
	h := hermes.Hermes{
		// Optional Theme
		// Theme: new(Default)
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: "Hermes",
			Link: "https://example-hermes.com/",
			// Optional product logo
			Logo: "http://www.duchess-france.org/wp-content/uploads/2016/01/gopher.png",
		},
	}
	email := hermes.Email{
		Body: hermes.Body{
			Name: "Jon Snow",
			Intros: []string{
				"Welcome to Hermes! We're very excited to have you on board.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "To get started with Hermes, please click here:",
					Button: hermes.Button{
						Color: "#22BC66", // Optional action button color
						Text:  "Confirm your account",
						Link:  "https://hermes-example.com/confirm?token=d9729feb74992cc3482b350163a1a010",
					},
				},
			},
			Outros: []string{
				"Need help, or have questions? Just reply to this email, we'd love to help.",
			},
		},
	}

	// Generate an HTML email with the provided contents (for modern clients)
	emailBody, err := h.GenerateHTML(email)
	if err != nil {
		panic(err) // Tip: Handle error with something else than a panic ;)
	}

	// Generate the plaintext version of the e-mail (for clients that do not support xHTML)
	emailText, err := h.GeneratePlainText(email)
	if err != nil {
		panic(err) // Tip: Handle error with something else than a panic ;)
	}

	mg := mailgun.NewMailgun("mg.storiesincognito.org", "key-3d2e7518cd8fd1332f07f4f7013bf680", "key-3d2e7518cd8fd1332f07f4f7013bf680")
	message := mailgun.NewMessage(
		"support@storiesincognito.org",
		"Welcome to Stories Incognito",
		emailText,
		address)
	message.SetHtml(emailBody)
	resp, id, err := mg.Send(message)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ID: %s Resp: %s\n", id, resp)
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
		logrus.WithFields(logrus.Fields{
			"func": "IsAdmin - apikey",
		}).Info(err.Error())
		return false
	}
	var userID string
	err = keys.Get("apikey:"+apikey, &userID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func": "IsAdmin - userID",
		}).Info(err.Error())
		return false
	}
	u, err := user.Get(userID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func": "IsAdmin - email",
		}).Info(err.Error())
		return false
	}
	var foo string
	err = keys.Get("admin:"+u.Email, &foo)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func": "IsAdmin - key check",
		}).Info(err.Error())
		return false
	}
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
	apikey := utils.NewAPIKey()
	err = keys.Set("apikey:"+apikey, userID)
	if err != nil {
		return
	}

	// Set the cookie with the API key
	err = setCookie("apikey", apikey, c)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func": "SignIn",
		}).Info(err.Error())
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
	if err == nil {
		u, err2 := user.Get(userID)
		if err2 == nil {
			logrus.WithFields(logrus.Fields{
				"func": "GetUserIDFromCookie",
			}).Infof("email:%s userid:%s", u.Email, userID)
		}
	}
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
	cookies := sessions.Default(c)
	cookies.Set("continueon", c.Request.URL.String())
	err := cookies.Save()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"func": "SignInAndContinueOn",
		}).Info(err.Error())
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
