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
	"os/signal"
	"syscall"

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
	pages       map[string]*page // p.URL.Path for the string
	URL         *url.URL
	shutdown    chan os.Signal
	workerCount uint
}

// NewSiteMap returns a SiteMap initialized with the starting URL, path and the
// number of workers used when crawling the site. It also sets up signal
// handling which stops crawling for SIGTERM or SIGINT.
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
	sm := &SiteMap{
		pages:       map[string]*page{start.Path: newPage(start)},
		URL:         siteURL,
		workerCount: workerCount,
	}

	sm.shutdown = make(chan os.Signal, 2)
	signal.Notify(sm.shutdown, syscall.SIGINT, syscall.SIGTERM)
	return sm, nil
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
// number of workers, exiting when the process is completed or when a signal
// is received on the SiteMap shutdown channel.
func (sm *SiteMap) Start() error {
	// TODO setup performance tests to determine the best buffer sizes
	new := make(chan *page, sm.workerCount*2)
	visited := make(chan *page, sm.workerCount*2)

	c := newCrawler()
	for i := uint(0); i < sm.workerCount; i++ {
		c.crawl(new, visited)
	}

	for _, p := range sm.pages {
		if !p.visited {
			new <- p
		}
	}
	var visitCount int
	for {
		pageCount.Set(float64(len(sm.pages)))
		if visitCount < len(sm.pages) {
			select {
			case p := <-visited:
				visitCount++
				pagesVisited.Inc()
				toVisit := sm.addPages(p.links)
				go func() { // add to new without blocking processing of visited
					for _, p := range toVisit {
						new <- p
					}
				}()
			case sig := <-sm.shutdown:
				c.stop()
				return fmt.Errorf("received shutdown signal %s", sig)
			}
		} else if visitCount == len(sm.pages) {
			return nil
		}
	}
}

// addPages walks through the given site relative paths adding new pages for
// each path not already part of sm.Pages and returning those added as a list.
func (sm *SiteMap) addPages(links map[string]int) []*page {
	var pages []*page
	for path := range links {
		if _, ok := sm.pages[path]; !ok {
			u, err := url.Parse(path)
			if err != nil {
				log.Printf("failed to parse relative path %q from page link, all these paths should be prevetted", path)
				continue
			}
			p := newPage(sm.URL.ResolveReference(u))
			sm.pages[path] = p
			pages = append(pages, p)
		}
	}

	return pages
}
