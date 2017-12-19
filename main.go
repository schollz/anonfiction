package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/matcornic/hermes"
	"github.com/pkg/errors"
	"github.com/schollz/anonfiction/src/encrypt"
	"github.com/schollz/anonfiction/src/story"
	"github.com/schollz/anonfiction/src/topic"
	"github.com/schollz/anonfiction/src/user"
	"github.com/schollz/anonfiction/src/utils"
	strip "github.com/schollz/html-strip-tags-go"
	"github.com/schollz/jsonstore"
	"github.com/sirupsen/logrus"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"
)

var (
	port, mailgunAPIKey string
	keys                *jsonstore.JSONStore
	// Keys contain the "Validator", the "API Keys" and the "Admin" users. Sign in requests "Email". Server generates a UUID for that email address and stores in a key "uuid:Y" with the User ID as the value. An email is sent with a link, /register?key=Y where Y is UUID. When traversing the link, the server checks that the UUID is valid (it exists as a key "uuid:Y" in Validator) and gets the User ID value. If valid, it generates a API key and adds the User ID to the map (key: "apikey:X" with value User ID) and and sets a cookie containing the encrypted API key, and then deletes the UUID key. All things requiring authentication use the APIKey to determine if it is a valid user and get the and to identify the user by the User ID (each computer will be signed in unless the cookie is deleted). Signing out deletes the cookie and deletes the APIKey.

	// Basically:
	// 		UUID ensures that API keys can't be generated without requesting one first
	// 		Deleting UUID after registering ensures one email = one API key
	// 		Multiple API keys ensures one user can login multiple times and signing out does not affect logins

	// Keys also stores the admins. To add an admin simple put in a Key "admin:someemail@something.com" with a value "\"something\""

)

const (
	TopicDB          = "topics.db.json"
	UniversalMessage = template.HTML(`<a href="/promo" class="special-link">Get paid for your story!</a>`)
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

func firsttenwords(s string) string {
	s = strings.Replace(s, "&nbsp;", " ", -1)
	words := strings.Fields(strip.StripTags(s))
	if len(words) > 10 {
		words = words[:10]
	}
	return strings.Join(words, " ")
}

func main() {
	flag.StringVar(&port, "port", "3001", "port of server")
	flag.StringVar(&mailgunAPIKey, "mailgun", "", "mailgun private API key")
	flag.Parse()
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	store := sessions.NewCookieStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))
	router.SetFuncMap(template.FuncMap{
		"slugify":       slugify,
		"unslugify":     unslugify,
		"firsttenwords": firsttenwords,
	})
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")
	router.HEAD("/", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/rss.xml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/rss+xml", []byte(RSS()))
	})
	router.GET("/sitemap.xml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/xml", []byte(SiteMap()))
	})
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "landing.tmpl", MainView{
			Landing:         true,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
			InfoMessageHTML: UniversalMessage,
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
			id = strings.ToLower(strings.TrimSpace(actions[2]))
		}

		var err error
		var s story.Story
		var t topic.Topic
		var stories []story.Story
		var iNum int
		var nextStory, previousStory, nextTopic string
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
			storiesString := "stories"
			if strings.Contains(id, "reply-to") {
				storiesString = "replies"
			}
			c.HTML(http.StatusOK, "error.tmpl", MainView{
				IsAdmin:         IsAdmin(c),
				SignedIn:        IsSignedIn(c),
				InfoMessageHTML: template.HTML("No " + storiesString + " yet, <a href='/write?topic=" + id + "'>why don't you write one?</a>"),
				ErrorCode:       RandomQuote(),
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
		} else {
			nextTopic = topic.Next(TopicDB, s.Topic)
		}
		if iNum > 1 {
			previousStory = strconv.Itoa(iNum - 1)
		}
		t, _ = topic.Get(TopicDB, s.Topic)
		c.HTML(http.StatusOK, "read.tmpl", MainView{
			InfoMessageHTML: UniversalMessage,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
			Topic:           t,
			Story:           s,
			Next:            nextStory,
			NextTopic:       nextTopic,
			Previous:        previousStory,
			NumStory:        iNum,
			NumStories:      len(stories),
			Route:           action + "/" + id,
		})
	})
	router.GET("/write/*storyID", func(c *gin.Context) {
		chosenTopic := c.DefaultQuery("topic", "")
		var t topic.Topic
		if len(chosenTopic) > 0 {
			t, _ = topic.Get(TopicDB, unslugify(chosenTopic))
		}
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
			s = story.New(userID, t.Name, "", "", []string{})
		}
		if strings.Contains(chosenTopic, "reply-to") {
			s.Content.Update("Dear Editor,<break><break>")
		}
		c.HTML(http.StatusOK, "write.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Story:    s,
			Topics:   topics,
			Topic:    t,
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
		ShowProfile("", "", c)
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
			ShowProfile("", err.Error(), c)
		} else {
			ShowProfile("Story deleted.", "", c)
		}
	})
	router.GET("/topics", func(c *gin.Context) {
		topics, err := topic.Load(TopicDB)
		if err != nil {
			ShowError(err, c)
			return
		}
		c.HTML(http.StatusOK, "topics.tmpl", MainView{
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
			Topics:          topics,
			InfoMessageHTML: UniversalMessage,
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
		// for i, s := range stories {
		// 	u, _ := user.Get(s.UserID)
		// 	stories[i].UserID = u.Email
		// }
		log.Println(err)
		users, _ := user.All()
		// add email to the user ID
		c.HTML(http.StatusOK, "admin.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
			Stories:  stories,
			Users:    users,
		})
	})
	router.GET("/terms", func(c *gin.Context) {
		c.HTML(http.StatusOK, "terms.tmpl", MainView{
			InfoMessageHTML: UniversalMessage,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
		})
	})
	router.GET("/contact", func(c *gin.Context) {
		c.HTML(http.StatusOK, "contact.tmpl", MainView{
			InfoMessageHTML: UniversalMessage,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
		})
	})
	router.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "privacy.tmpl", MainView{
			InfoMessageHTML: UniversalMessage,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
		})
	})
	router.GET("/about", func(c *gin.Context) {
		c.HTML(http.StatusOK, "contact.tmpl", MainView{
			InfoMessageHTML: UniversalMessage,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
		})
	})
	router.GET("/promo", func(c *gin.Context) {
		c.HTML(http.StatusOK, "promo.tmpl", MainView{
			IsAdmin:  IsAdmin(c),
			SignedIn: IsSignedIn(c),
		})
	})
	router.GET("/guestbook", func(c *gin.Context) {
		c.HTML(http.StatusOK, "guestbook.tmpl", MainView{
			InfoMessageHTML: UniversalMessage,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
		})
	})
	router.GET("/stories", func(c *gin.Context) {
		topics, _ := topic.Load(TopicDB)
		stories, _ := story.ListPublished()
		s := make([]story.Story, len(stories))
		t := make([]topic.Topic, len(topics))
		si := 0
		ti := 0
		for _, topic := range topics {
			topic.NumberOfStories = 1
			fmt.Println(strings.Contains(strings.ToLower(topic.Name), "reply to"))
			if strings.Contains(strings.ToLower(topic.Name), "reply to") {
				continue
			}
			t[ti] = topic
			for _, story := range stories {
				if topic.Name == story.Topic {
					s[si] = story
					s[si].ID = strconv.Itoa(t[ti].NumberOfStories)
					t[ti].NumberOfStories++
					si++
				}
			}
			ti++
		}
		s = s[:si]
		t = t[:ti]
		c.HTML(http.StatusOK, "all.tmpl", MainView{
			Stories:         s,
			Topics:          t,
			IsAdmin:         IsAdmin(c),
			SignedIn:        IsSignedIn(c),
			InfoMessageHTML: UniversalMessage,
		})
	})
	router.GET("/download/:name", func(c *gin.Context) {
		if !IsSignedIn(c) {
			SignInAndContinueOn(c)
			return
		}
		userID, _ := GetUserIDFromCookie(c)
		s, err := story.ListByUser(userID)
		if err != nil {
			ShowError(err, c)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"stories": s,
		})
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(302, "/static/img/meta/favicon.ico")
	})
	router.POST("/write/*foo", handlePOSTStory)
	router.POST("/login", handlePOSTSignup)
	router.POST("/profile", handlePOSTProfile)
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
	NextTopic       string
	Previous        string
	TrixAttr        template.HTMLAttr
	Route           string
	User            user.User
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
		// isNewStory := false
		if err != nil {
			s = story.New(userID, form.Topic, "", "", []string{})
			s.ID = form.StoryID
			// isNewStory = true
		}
		s.Content.Update(form.Content)
		s.Topic = form.Topic
		s.Keywords = keywords
		s.Description = form.Description
		if form.Published == "on" {
			s.DatePublished = time.Now()
			s.Published = true
		} else {
			s.Published = false
		}
		if IsAdmin(c) {
			err = s.Save()
			// allow to save anonymous story!
			// } else if !isNewStory && userID == user.AnonymousUserID() {
			// 	err = errors.New("cannot update an anonymous story")
		} else if userID != s.UserID && userID != user.AnonymousUserID() {
			err = errors.New("cannot update someone elses story")
		} else {
			err = s.Save()
		}
		var infoMessage, errorMessage string
		if err != nil {
			err = errors.Wrap(err, "story not submitted")
			errorMessage = err.Error()
		} else {
			infoMessage = fmt.Sprintf("Story updated. Read it at <a href='/read/story/%s' class='washed-red' target='_blank'>/read/story/%s</a>.", s.ID, s.ID)
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

func handlePOSTProfile(c *gin.Context) {
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
			ShowError(err, c)
			return
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

		err = user.Update(userID, form.Email, form.Language, form.Digest == "on")
		if err != nil {
			ShowProfile("", err.Error(), c)
		} else {
			ShowProfile("User updated.", "", c)
		}
	} else {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			ErrorMessage: err.Error(),
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
		if mailgunAPIKey != "" {
			sendEmail(form.Email, uuid)
			c.HTML(http.StatusOK, "login.tmpl", MainView{
				InfoMessage: "Check your email for the link to login.",
				IsAdmin:     IsAdmin(c),
				SignedIn:    IsSignedIn(c),
			})
		} else {
			c.HTML(http.StatusOK, "login.tmpl", MainView{
				InfoMessageHTML: template.HTML("<a class='special-link' href='/login?key=" + uuid + "'>Click here to login</a>"),
				IsAdmin:         IsAdmin(c),
				SignedIn:        IsSignedIn(c),
			})
		}
	} else {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{
			ErrorMessage: err.Error(),
		})
	}
}

func sendEmail(address, key string) {
	// Configure hermes by setting a theme and your product info
	h := hermes.Hermes{
		// Optional Theme
		// Theme: new(Default)
		Product: hermes.Product{
			// Appears in header & footer of e-mails
			Name: "anonfiction team",
			Link: "https://www.anonfiction.com",
			// Optional product logo
			Logo: "https://www.anonfiction.com/static/img/books2.png",
		},
	}
	email := hermes.Email{
		Body: hermes.Body{
			Title: "Welcome to anonfiction!",
			// Intros: []string{
			// 	"Welcome to Stories Incognito!",
			// },
			Actions: []hermes.Action{
				{
					Instructions: "To login, please click here:",
					Button: hermes.Button{
						Color: "#00449e", // Optional action button color
						Text:  "Log In",
						Link:  "https://www.anonfiction.com/login?key=" + key,
					},
				},
			},
			Outros: []string{
				"Note: This link will only work once. Feel free to request new ones though!",
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

	mg := mailgun.NewMailgun("anonfiction.com", mailgunAPIKey, mailgunAPIKey)
	message := mailgun.NewMessage(
		"support@anonfiction.com",
		"anonfiction sign in ("+time.Now().Format("Jan 2 15:04")+")",
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

func ShowProfile(infoMessage, errorMessage string, c *gin.Context) {
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
	u, _ := user.Get(userID)
	c.HTML(http.StatusOK, "profile.tmpl", MainView{
		IsAdmin:      IsAdmin(c),
		SignedIn:     IsSignedIn(c),
		Stories:      stories,
		User:         u,
		InfoMessage:  infoMessage,
		ErrorMessage: errorMessage,
	})
}
