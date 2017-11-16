package main

import (
	"flag"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
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
		c.HTML(http.StatusOK, "write.tmpl", MainView{})
	})
	router.GET("/read", func(c *gin.Context) {
		c.HTML(http.StatusOK, "read.tmpl", MainView{})
	})
	router.GET("/archive", func(c *gin.Context) {
		c.HTML(http.StatusOK, "archive.tmpl", MainView{})
	})
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.tmpl", MainView{})
	})
	router.GET("/signup", func(c *gin.Context) {
		c.HTML(http.StatusOK, "signup.tmpl", MainView{})
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(302, "/static/img/meta/favicon.ico")
	})
	router.Run(":" + port)
}

type MainView struct {
	Title   string
	Landing bool
}

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "landing.tmpl", MainView{
		Landing: true,
	})
}
