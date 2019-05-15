package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gen "github.com/gorilla/feeds"  // the feed generator
	par "github.com/mmcdole/gofeed" // the feed parser
)

const (
	userAgent string = "HNcomments/0.1"
	urlBase   string = "https://news.ycombinator.com/rss"
)

var (
	fTimeout = flag.Int("timeout", 60, "http client timeout in seconds")
	fOutput  = flag.String("output", "comments.rss", "file path to write RSS feed to")
	fDebug   = flag.Bool("debug", false, "debug messages")
)

func main() {
	flag.Parse()

	//b, err := testFeed()
	b, err := downloadFeed(urlBase)
	if err != nil {
		panic(err)
	}
	defer b.Close()

	f, err := parseFeed(b)
	if err != nil {
		panic(err)
	}

	newf, err := createCommentFeed(f)
	if err != nil {
		panic(err)
	}

	err = writeFeed(newf, *fOutput)
	if err != nil {
		panic(err)
	}
}

func logMsg(msg string, args ...interface{}) {
	if *fDebug {
		log.Printf(msg+"\n", args...)
	}
}

//func testFeed() (io.ReadCloser, error) {
//b, err := os.Open("tmp/rss")
//if err != nil {
//return nil, err
//}
//return b, nil
//}

func downloadFeed(u string) (io.ReadCloser, error) {
	logMsg("Downloading feed...")
	c := http.Client{
		Timeout: time.Duration(*fTimeout) * time.Second,
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("bad response status: %s", res.Status)
	}
	return res.Body, nil
}

func parseFeed(b io.Reader) (*par.Feed, error) {
	logMsg("Parsing...")
	p := par.NewParser()
	f, err := p.Parse(b)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func createCommentFeed(f *par.Feed) (*gen.Feed, error) {
	newf := &gen.Feed{
		Title:       f.Title,
		Link:        &gen.Link{Href: f.Link},
		Description: f.Description,
		Copyright:   "Copyright © 2005–2018 Y Combinator, LLC. All rights reserved.",
		Updated:     time.Now().UTC(),
	}

	for _, i := range f.Items {
		parts := strings.Split(i.Description, "\"")
		if len(parts) != 3 {
			logMsg("Failed to parse comment link from item: %#v", i)
			continue
		}
		l := parts[1]
		newf.Add(&gen.Item{
			Title:       i.Title,
			Link:        &gen.Link{Href: l},
			Created:     i.PublishedParsed.UTC(),
			Description: fmt.Sprintf("Submitted link: %s", i.Link),
		})
	}

	return newf, nil
}

func writeFeed(newf *gen.Feed, p string) error {
	p = filepath.Clean(p)
	logMsg("Writing feed to: %s", p)
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	err = newf.WriteRss(f)
	if err != nil {
		return err
	}
	return nil
}
