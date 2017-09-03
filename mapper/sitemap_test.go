package mapper

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
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
		case test.wantURL != nil && len(sm.Pages) != 1:
			t.Errorf("Test %q - got %d pages, want 1", test.name, len(sm.Pages))
		case test.wantURL != nil && !reflect.DeepEqual(sm.URL, test.wantURL):
			t.Errorf("Test %q - got url %q, want %q", test.name, sm.URL, test.wantURL)
		case test.wantStartURL != nil && !reflect.DeepEqual(sm.Pages[test.wantStartURL.Path].URL, test.wantStartURL):
			t.Errorf("Test %q - got start url %q, want %q", test.name, sm.Pages[test.wantStartURL.Path].URL, test.wantStartURL)
		}
	}
}

func TestAddPages(t *testing.T) {
	sm, err := NewSiteMap("http://testsite.com", 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(sm.Pages) != 1 {
		t.Fatalf("Expected to start with a single page")
	}

	set1 := map[string]int{"/test1": 1, "/test1/page1": 2}
	newPages := sm.addPages(set1)

	if got, want := len(newPages), len(set1); got != want {
		t.Errorf("Got %d new pages, want %d", got, want)
	}
	if got, want := len(sm.Pages), len(set1)+1; got != want {
		t.Errorf("Got sm.Pages length %d, want %d", got, want)
	}
	for _, p := range newPages {
		path := p.URL.Path
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
	if got, want := len(sm.Pages), len(set2)+1; got != want {
		t.Errorf("Got sm.Pages length %d, want %d", got, want)
	}
	for _, p := range newPages {
		path := p.URL.Path
		if _, ok := set2[path]; !ok {
			t.Errorf("Path %q is an unexpected new page", path)
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
		"/": &page{
			Links:   map[string]int{"/hello-world": 1, "/values": 1, "/variables": 1},
			URL:     baseURL,
			Visited: true,
		},
		"/hello-world": &page{
			Links:   map[string]int{"/": 1, "/values": 1},
			URL:     baseURL.ResolveReference(&url.URL{Path: "/hellow-world"}),
			Visited: true,
		},
		"/values": &page{
			Links:   map[string]int{"/": 1, "/variables": 1},
			URL:     baseURL.ResolveReference(&url.URL{Path: "/values"}),
			Visited: true,
		},
		"/variables": &page{
			Links:   map[string]int{"/": 1, "/constants": 1},
			URL:     baseURL.ResolveReference(&url.URL{Path: "/variables"}),
			Visited: true,
		},
		"/constants": &page{
			Links:   map[string]int{},
			Broken:  true,
			URL:     baseURL.ResolveReference(&url.URL{Path: "/constants"}),
			Visited: true,
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

	if got, want := len(sm.Pages), len(wantPages); got != want {
		t.Errorf("Got %d pages, want %d", got, want)
	}
	for path, page := range sm.Pages {
		wantPage, ok := wantPages[path]
		if !ok {
			t.Errorf("Got unwanted path %q", path)
			continue
		}
		if !reflect.DeepEqual(page.Links, wantPage.Links) {
			t.Errorf("Path %q got links\n%v\nwant links\n%v\n", path, page.Links, wantPage.Links)
		}
	}
}
