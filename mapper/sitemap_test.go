package mapper

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestNewSiteMap(t *testing.T) {
	tests := []struct {
		name         string
		startPage    string
		workerCount  uint
		wantErr      bool
		wantStartURL *url.URL
		wantURL      *url.URL
	}{
		{
			name:         "basic",
			startPage:    "http://mysite.com/",
			workerCount:  2,
			wantStartURL: &url.URL{Scheme: "http", Host: "mysite.com", Path: "/"},
			wantURL:      &url.URL{Scheme: "http", Host: "mysite.com"},
		},
		{
			name:         "no trailing /",
			startPage:    "http://mysite.com",
			workerCount:  2,
			wantStartURL: &url.URL{Scheme: "http", Host: "mysite.com", Path: "/"},
			wantURL:      &url.URL{Scheme: "http", Host: "mysite.com"},
		},
		{
			name:         "no scheme",
			startPage:    "mysite.com",
			workerCount:  2,
			wantStartURL: &url.URL{Scheme: "http", Host: "mysite.com", Path: "/"},
			wantURL:      &url.URL{Scheme: "http", Host: "mysite.com"},
		},
		{
			name:         "/startpage",
			startPage:    "https://mysite.com/startpage",
			workerCount:  4,
			wantStartURL: &url.URL{Scheme: "https", Host: "mysite.com", Path: "/startpage"},
			wantURL:      &url.URL{Scheme: "https", Host: "mysite.com"},
		},
		{
			name:         "/startpage/subpage",
			startPage:    "https://mysite.com/startpage/subpage",
			workerCount:  4,
			wantStartURL: &url.URL{Scheme: "https", Host: "mysite.com", Path: "/startpage/subpage"},
			wantURL:      &url.URL{Scheme: "https", Host: "mysite.com"},
		},
		{
			name:         "no scheme and /startpage",
			startPage:    "mysite.com/startpage",
			workerCount:  4,
			wantStartURL: &url.URL{Scheme: "http", Host: "mysite.com", Path: "/startpage"},
			wantURL:      &url.URL{Scheme: "http", Host: "mysite.com"},
		},
		{
			name:        "0 workers",
			startPage:   "http://mysite.com",
			workerCount: 0,
			wantErr:     true,
		},
		{
			name:        "invalid url",
			startPage:   "http://`mysite.com",
			workerCount: 4,
			wantErr:     true,
		},
	}

	for _, test := range tests {
		sm, err := NewSiteMap(test.startPage, test.workerCount)

		switch {
		case err != nil && !test.wantErr:
			t.Errorf("Test %q - got error, want nil: %v", test.name, err)
		case err == nil && test.wantErr:
			t.Errorf("Test %q - got nil, want error", test.name)
		case test.wantURL != nil && len(sm.pages) != 1:
			t.Errorf("Test %q - got %d pages, want 1", test.name, len(sm.pages))
		case test.wantURL != nil && !reflect.DeepEqual(sm.URL, test.wantURL):
			t.Errorf("Test %q - got url %q, want %q", test.name, sm.URL, test.wantURL)
		case test.wantStartURL != nil && !reflect.DeepEqual(sm.pages[test.wantStartURL.Path].url, test.wantStartURL):
			t.Errorf("Test %q - got start url %q, want %q", test.name, sm.pages[test.wantStartURL.Path].url, test.wantStartURL)
		}
	}
}

func TestAddPages(t *testing.T) {
	sm, err := NewSiteMap("http://testsite.com", 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(sm.pages) != 1 {
		t.Fatalf("Expected to start with a single page")
	}

	set1 := map[string]int{"/test1": 1, "/test1/page1": 2}
	newPages := sm.addPages(set1)

	if got, want := len(newPages), len(set1); got != want {
		t.Errorf("Got %d new pages, want %d", got, want)
	}
	if got, want := len(sm.pages), len(set1)+1; got != want {
		t.Errorf("Got sm.Pages length %d, want %d", got, want)
	}
	for _, p := range newPages {
		path := p.url.Path
		if _, ok := set1[path]; !ok {
			t.Errorf("Path %q is an unexpected new page", path)
		}
	}

	set2 := set1
	set2["/test2"] = 1
	set2["/test2/page"] = 1
	newPages = sm.addPages(set2)

	if got, want := len(newPages), 2; got != want {
		t.Errorf("Got %d new pages, want %d", got, want)
	}
	if got, want := len(sm.pages), len(set2)+1; got != want {
		t.Errorf("Got sm.Pages length %d, want %d", got, want)
	}
	for _, p := range newPages {
		path := p.url.Path
		if _, ok := set2[path]; !ok {
			t.Errorf("Path %q is an unexpected new page", path)
		}
	}
}

func TestHandleSignals(t *testing.T) {
	sm, err := NewSiteMap("http://localhost", 2)
	if err != nil {
		t.Fatal(err)
	}
	pid := os.Getpid()
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		if err := syscall.Kill(pid, sig); err != nil {
			t.Fatalf("syscall Kill signal %s failed: %v", sig, err)
		}
		select {
		case <-sm.shutdown:
		case <-time.After(20 * time.Millisecond):
			t.Errorf("Received no shutdown command for signal %s", sig)
		}
	}
}

func TestStart(t *testing.T) {
	server := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	wantPages := map[string]*page{
		"/": {
			links:   map[string]int{"/hello-world": 1, "/values": 1, "/variables": 1},
			url:     baseURL,
			visited: true,
		},
		"/hello-world": {
			links:   map[string]int{"/": 1, "/values": 1},
			url:     baseURL.ResolveReference(&url.URL{Path: "/hellow-world"}),
			visited: true,
		},
		"/values": {
			links:   map[string]int{"/": 1, "/variables": 1},
			url:     baseURL.ResolveReference(&url.URL{Path: "/values"}),
			visited: true,
		},
		"/variables": {
			links:   map[string]int{"/": 1, "/constants": 1},
			url:     baseURL.ResolveReference(&url.URL{Path: "/variables"}),
			visited: true,
		},
		"/constants": {
			links:   map[string]int{},
			broken:  true,
			url:     baseURL.ResolveReference(&url.URL{Path: "/constants"}),
			visited: true,
		},
	}

	u, err := url.Parse(server.URL + "/hello-world")
	if err != nil {
		t.Fatal(err)
	}

	sm, err := NewSiteMap(u.String(), 2)
	if err != nil {
		t.Fatal(err)
	}

	if err := sm.Start(); err != nil {
		t.Errorf("Start error: %v", err)
	}

	if got, want := len(sm.pages), len(wantPages); got != want {
		t.Errorf("Got %d pages, want %d", got, want)
	}
	for path, page := range sm.pages {
		wantPage, ok := wantPages[path]
		if !ok {
			t.Errorf("Got unwanted path %q", path)
			continue
		}
		if !reflect.DeepEqual(page.links, wantPage.links) {
			t.Errorf("Path %q got links\n%v\nwant links\n%v\n", path, page.links, wantPage.links)
		}
	}
}
