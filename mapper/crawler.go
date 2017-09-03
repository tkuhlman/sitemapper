package mapper

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/net/html"
)

const clientTimeout = 5 * time.Second

type crawler struct {
	client       *http.Client
	stopChannels []chan bool
}

// newCrawler returns a crawler using the default http client with a faster
// timeout.
func newCrawler() *crawler {
	c := http.DefaultClient
	c.Timeout = clientTimeout
	return &crawler{client: c}
}

// crawl start a go routine that pulls pages from the new channel visits them
// and puts the result onto the finished channel. c.stop will halt all of the
// go routines.
func (c *crawler) crawl(new <-chan *page, finished chan<- *page) {
	stop := make(chan bool, 1)
	c.stopChannels = append(c.stopChannels, stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case p := <-new:
				c.visit(p)
				// TODO consider adding a histogram for stats on visit timing
				finished <- p
			}
		}
	}()
}

// get issues an HTTP GET to the given URL using the crawler client. Any
// non-2XX status codes are considered an error. The body of the response
// is returned on success.
func (c *crawler) get(url string) (io.ReadCloser, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("Status code %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// stop sends a signal to each go routine doing crawling to stop any activity.
func (c *crawler) stop() {
	for _, c := range c.stopChannels {
		c <- true
	}
}

// Crawler connects to the page and extract all the links populating p.Links.
// Any non-200 response code will result in p.Broken being set to true.
func (c *crawler) visit(p *page) {
	p.Visited = true
	body, err := c.get(p.URL.String())
	if err != nil {
		p.Broken = true
		p.err = err
		return
	}

	p.addLinks(extractLinks(body))
}

// extractLinks parses an html page and returns the href for all of the
// anchor tags.
func extractLinks(body io.ReadCloser) []string {
	defer body.Close()
	var links []string
	tokens := html.NewTokenizer(body)
	for {
		tt := tokens.Next()
		switch tt {
		case html.ErrorToken:
			return links
		case html.StartTagToken:
			token := tokens.Token()
			if token.Data == "a" {
				for _, a := range token.Attr {
					if a.Key == "href" {
						links = append(links, a.Val)
					}
				}
			}
		}
	}
}
