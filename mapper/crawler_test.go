package mapper

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestCrawl(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer server.Close()

	u, err := url.Parse(server.URL + "hello-world")
	if err != nil {
		t.Fatal(err)
	}
	p := newPage(u)

	new := make(chan *page, 2)
	finished := make(chan *page, 2)

	new <- p

	c := newCrawler()
	c.crawl(new, finished)
	time.Sleep(10 * time.Millisecond) // needed because I block on finished below
	c.stop()

	got := <-finished
	if got.URL != p.URL {
		t.Errorf("Got url %q, want %q", got.URL, p.URL)
	}
	if !got.Visited {
		t.Error("Finished page unvisited")
	}
}

func TestExtractLinks(t *testing.T) {
	wantLinks := []string{
		"./",
		"http://play.golang.org/p/2C7wwJ6nxG",
		"values",
		"https://twitter.com/mmcgrana",
		"mailto:mmcgrana@gmail.com",
		"https://github.com/mmcgrana/gobyexample/blob/master/examples/hello-world",
		"https://github.com/mmcgrana/gobyexample#license",
	}

	f, err := os.Open("testdata/hello-world")
	if err != nil {
		t.Fatal(err)
	}

	links := extractLinks(f)

	if !reflect.DeepEqual(links, wantLinks) {
		t.Errorf("Got links\n%v\nwant links\n%v\n", links, wantLinks)
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{
			path: "/hello-world",
		},
		{
			path:    "/goodbye-world",
			wantErr: true,
		},
	}

	server := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer server.Close()

	c := newCrawler()

	for _, test := range tests {
		body, err := c.get(server.URL + test.path)
		if test.wantErr {
			if err == nil {
				t.Errorf("Test path %q - got nil want error", test.path)
			}
		} else {
			if err != nil {
				t.Errorf("Test path %q - got error want nil: %v", test.path, err)
			}
			if body == nil {
				t.Errorf("Test path %q - got nil body", test.path)
			}
		}
	}
}

func TestVisit(t *testing.T) {
	wantLinks := map[string]int{"/": 1, "/values": 1}

	server := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer server.Close()

	c := newCrawler()
	u, err := url.Parse(server.URL + "/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	p := newPage(u)

	c.visit(p)
	if !reflect.DeepEqual(p.Links, wantLinks) {
		t.Errorf("Got links %v, want %v", p.Links, wantLinks)
	}
}
