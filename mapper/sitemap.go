// Package mapper contains the data structures for a site map, a simple directed
// graph, and the tooling to create one by crawling a website.
package mapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	pageCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "page_count",
		Help: "Current count of pages in the site being mapped",
	})
	pagesVisited = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pages_visited",
		Help: "The number of pages for which an HTTP GET has been attempted.",
	})
)

func init() {
	prometheus.MustRegister(pageCount)
	prometheus.MustRegister(pagesVisited)
}

// SiteMap is the data structure in which a mapping of a website is built.
type SiteMap struct {
	Pages       map[string]*page // p.URL.Path for the string
	URL         *url.URL
	shutdown    chan string
	workerCount uint
}

// NewSiteMap returns a SiteMap initialized with the starting URL, path and the
// number of workers used when crawling the site.
func NewSiteMap(startPage string, workerCount uint) (*SiteMap, error) {
	if workerCount < 1 {
		return nil, errors.New("workerCount for a SiteMap must be > 0")
	}
	start, err := url.Parse(startPage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page %q: %v", startPage, err)
	}

	if start.Scheme == "" {
		log.Printf("No URL scheme specified using 'http'")
		start, err = url.Parse("http://" + startPage)
		if err != nil {
			return nil, fmt.Errorf("failed to parse page %q: %v", "http://"+startPage, err)
		}
	}
	siteURL := &url.URL{
		Scheme: start.Scheme,
		Host:   start.Host,
	}
	if start.Path == "" {
		start.Path = "/"
	}
	return &SiteMap{
		Pages:       map[string]*page{start.Path: newPage(start)},
		URL:         siteURL,
		shutdown:    make(chan string),
		workerCount: workerCount,
	}, nil
}

// HandleSignals reads any signals from the channel and shuts down a
// running site crawl as appropriate. This function should run in a
// go routine and signals should be registred with signal.Notify
func (sm *SiteMap) HandleSignals(signals <-chan os.Signal) {
	sig := <-signals
	sm.shutdown <- sig.String()
	// TODO add tests
}

// ServeHTTP implments the http.Handler interface responding with sm marshaled
// as JSON.
func (sm *SiteMap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	if err := enc.Encode(sm); err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal sitemap as JSON: %v", err), http.StatusInternalServerError)
	}
}

// Start begins crawling a website with the starting URL using the assigned
// number of workers, exiting when the process is completed.
func (sm *SiteMap) Start() error {
	// TODO setup performance tests to determine the best buffer sizes
	new := make(chan *page, sm.workerCount*2)
	visited := make(chan *page, sm.workerCount*2)

	c := newCrawler()
	for i := uint(0); i < sm.workerCount; i++ {
		c.crawl(new, visited)
	}

	for _, p := range sm.Pages {
		if !p.Visited {
			new <- p
		}
	}
	var visitCount int
	for {
		pageCount.Set(float64(len(sm.Pages)))
		if visitCount < len(sm.Pages) {
			select {
			case p := <-visited:
				visitCount++
				pagesVisited.Inc()
				toVisit := sm.addPages(p.Links)
				go func() { // add to new without blocking processing of visited
					for _, p := range toVisit {
						new <- p
					}
				}()
			case msg := <-sm.shutdown:
				c.stop()
				return fmt.Errorf("received shutdown signal %s", msg)
			}
		} else if visitCount == len(sm.Pages) {
			return nil
		}
	}
}

// addPages walks through the given site relative paths adding new pages for
// each path not already part of sm.Pages and returning those added as a list.
func (sm *SiteMap) addPages(links map[string]int) []*page {
	var pages []*page
	for path := range links {
		if _, ok := sm.Pages[path]; !ok {
			u, err := url.Parse(path)
			if err != nil {
				log.Printf("failed to parse relative path %q from page link, all these paths should be prevetted", path)
				continue
			}
			p := newPage(sm.URL.ResolveReference(u))
			sm.Pages[path] = p
			pages = append(pages, p)
		}
	}

	return pages
}

// TODO add the web handler, it will need to display little info until things are done, at least until concurrency is fully handled.
