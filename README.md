<p align="center">
<img
    src="/static/img/anonfiction.png"
    width="450px" border="0" alt="anonfiction">
<br>
<a href="https://travis-ci.org/schollz/anonfiction"><img src="https://travis-ci.org/schollz/anonfiction.svg?branch=master" alt="Build Status"></a>
<a href="https://github.com/schollz/anonfiction/releases/latest"><img src="https://img.shields.io/badge/version-0.1.0-brightgreen.svg?style=flat-square" alt="Version"></a>
<a href="https://goreportcard.com/report/github.com/schollz/anonfiction"><img src="https://goreportcard.com/badge/github.com/schollz/anonfiction" alt="Go Report Card"></a>

<p align="center">A homemade CMS designed for writing and publishing sincere and honest nonfiction stories.</p>

This repository powers [www.anonfiction.com](https://www.anonfiction.com). Your welcome to fork this project, or steal ideas from it, or contribute to it. It basically contains a fully-fledged CMS (content-management system) with passwordless authentication. I needed something similar to Wordpress and Ghost, but I wanted more control in the layout and the interface so I decided to write it myself. Its written in Go with the [Gin web framework](https://github.com/gin-gonic/gin) and the CSS framework is in [Tachyon](http://www.tachyons.io). The database uses [BoltDB via @asdine's storm interface](https://github.com/asdine/storm) and my simple [schollz/jsonstore](https://github.com/schollz/jsonstore). All changes to every story are saved via [schollz/versionedtext](github.com/schollz/versionedtext). The built-in editor comes from [basecamp/trix](https://github.com/basecamp/trix).

## Why?

I decided to make an effort to make a place on the Internet that is more positive and reflective. This place would be where you can present your own story, anonymously and sincerely. A place where you don't have to be an MFA to present a good story. I wrote [a blog post with more of my reasoning](https://schollz.github.io/anonfiction).

## Install

First install the dependencies using Go:

```
go get -u -v github.com/schollz/anonfiction
```

Then `cd` into the `$GOPATH` and build:

```
cd $GOPATH/src/github.com/schollz/anonfiction
go build
./anonfiction
```

The passwordless login requires an email server in production, but it is disabled by default (so anyone can login to anything). If you'd like to use it, I suggest getting [mailgun](https://www.mailgun.com/).

## Contributing

Pull requests are welcome. Feel free to...

- Revise documentation
- Add new features
- Fix bugs
- Suggest improvements

## License

MIT
