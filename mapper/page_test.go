package mapper

import (
	"net/url"
	"reflect"
	"testing"
)

var (
	testPageURL = &url.URL{Scheme: "http", Host: "testhost.com", Path: "/test"}
)

func TestAddLinks(t *testing.T) {
	testLinks := []string{
		"http://testhost.com",
		"http://testhost.com/test",
		"http://testhost.com/test1",
		"test1",
	}
	wantLinks := map[string]int{
		"/":      1,
		"/test1": 2,
	}

	p := newPage(testPageURL)
	p.addLinks(testLinks)

	if !reflect.DeepEqual(p.Links, wantLinks) {
		t.Errorf("Got links %+v, want %+v", p.Links, wantLinks)
	}
}

func TestFilterLink(t *testing.T) {
	tests := []struct {
		link     string
		want     string
		wantOkay bool
	}{
		{
			link:     "http://testhost.com",
			want:     "/",
			wantOkay: true,
		},
		{
			link:     "http://testhost.com/",
			want:     "/",
			wantOkay: true,
		},
		{
			link:     "http://testhost.com/test",
			want:     "",
			wantOkay: false,
		},
		{
			link:     "http://testhost.com/test/",
			want:     "/test/", // note the trailing slash technically makes it a different page
			wantOkay: true,
		},
		{
			link:     "http://testhost.com/test2",
			want:     "/test2",
			wantOkay: true,
		},
		{
			link:     "http://testhost.com/test2#toc",
			want:     "/test2",
			wantOkay: true,
		},
		{
			link:     "http://testhost.com/test2?option1=value1",
			want:     "/test2",
			wantOkay: true,
		},
		{
			link:     "http://testhost.com/test/page1",
			want:     "/test/page1",
			wantOkay: true,
		},
		{
			link:     "test/page1",
			want:     "/test/page1",
			wantOkay: true,
		},
		{
			link:     "test/page1/",
			want:     "/test/page1/",
			wantOkay: true,
		},
		{
			link:     "mailto:me@here.edu",
			want:     "",
			wantOkay: false,
		},
		{
			link:     "gopher://testhost.com",
			want:     "",
			wantOkay: false,
		},
	}

	p := newPage(testPageURL)
	for _, test := range tests {
		link, okay := p.filterLink(test.link)
		switch {
		case okay != test.wantOkay:
			t.Errorf("Test %q - got okay %t, want %t", test.link, okay, test.wantOkay)
		case link != test.want:
			t.Errorf("Test %q - got link %q, want %q", test.link, link, test.want)
		}
	}
}
