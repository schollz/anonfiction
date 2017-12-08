package main

import (
	"bytes"
	"html/template"
	"log"
	"time"

	"github.com/schollz/anonfiction/src/story"
	"github.com/schollz/anonfiction/src/topic"
)

const rssTemplate = `
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
	<atom:link href="https://www.anonfiction.com/rss.xml" rel="self" type="application/rss+xml" />
	<title>Stories Incognito</title>
	<link>https://www.anonfiction.com</link>
	<description></description>
	<generator></generator>
	<language>eng</language>
	<managingEditor>editor@anonfiction.com (Stories Incognito Editors)</managingEditor>
	<webMaster>web@anonfiction.com (Web Guru)</webMaster>
	<copyright>Copyright 2017 Stories Incognito</copyright>
	<lastBuildDate>{{ .Date.Format "Mon, 02 Jan 2006 15:04:05 -0700" }}</lastBuildDate>
	{{ range .Stories }}<item>
		<title>{{ .Topic }}</title>
		<link>https://www.anonfiction.com/read/story/{{ .ID }}</link>
		<pubDate>{{ .DatePublished.Format "Mon, 02 Jan 2006 15:04:05 -0700" }}</pubDate>
		<guid>https://www.anonfiction.com/read/story/{{ .ID }}</guid>
		<description>{{ .Description }}</description>
	</item>{{ end }}
</channel>
</rss>
`

func RSS() string {
	funcMap := template.FuncMap{
		"slugify":   slugify,
		"unslugify": unslugify,
	}

	// Create a template, add the function map, and parse the text.
	tmpl, err := template.New("rss").Funcs(funcMap).Parse(rssTemplate)
	if err != nil {
		log.Fatalf("parsing: %s", err)
	}

	type RSSData struct {
		Stories []story.Story
		Date    time.Time
	}

	s, _ := story.ListPublished()
	rss := RSSData{
		Date:    time.Now(),
		Stories: s,
	}
	// Run the template to verify the output.
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, rss)
	if err != nil {
		log.Fatalf("execution: %s", err)
	}
	return tpl.String()
}

const siteMapTemplate = `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
{{ range .Topics }}<url>
<loc>https://www.anonfiction.com/read/topic/{{ .Name | slugify }}</loc>
<lastmod>{{ $.Date.Format "2006-01-02T15:04:05-07:00" }}</lastmod>
<changefreq>monthly</changefreq>
<priority>1</priority>
</url>{{ end }}
{{ range .Stories }}<url>
<loc>https://www.anonfiction.com/read/story/{{ .ID }}</loc>
<lastmod>{{ .DatePublished.Format "2006-01-02T15:04:05-07:00" }}</lastmod>
<changefreq>monthly</changefreq>
<priority>0.75</priority>
</url>{{ end }}
</urlset>`

func SiteMap() string {
	funcMap := template.FuncMap{
		"slugify":   slugify,
		"unslugify": unslugify,
	}

	// Create a template, add the function map, and parse the text.
	tmpl, err := template.New("sitemap").Funcs(funcMap).Parse(siteMapTemplate)
	if err != nil {
		log.Fatalf("parsing: %s", err)
	}

	type Data struct {
		Stories []story.Story
		Topics  []topic.Topic
		Date    time.Time
	}

	s, _ := story.ListPublished()
	t, _ := topic.Load(TopicDB)
	data := Data{
		Date:    time.Now(),
		Stories: s,
		Topics:  t,
	}
	// Run the template to verify the output.
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		log.Fatalf("execution: %s", err)
	}
	return tpl.String()
}
