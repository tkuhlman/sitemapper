package mapper

import "net/url"

// page represents a single page within the site map. It tracks the links
// to the from this page to other paths on the same site.
type page struct {
	broken  bool
	links   map[string]int // string is the relative path, int a count of the number of links
	url     *url.URL
	visited bool
	err     error
}

// newPage returns a new unvisited page.
func newPage(url *url.URL) *page {
	return &page{links: map[string]int{}, url: url}
}

// addLinks will filter out any self links and links outside the base site
// then add what remains to p.Links
func (p *page) addLinks(links []string) {
	for _, link := range links {
		if linkPath, ok := p.filterLink(link); ok {
			p.links[linkPath]++
		}
	}
}

// filterLink will normalize the link url, filter out self links and links to
// a different host and then return the relative path portion of the URL.
// If a link is filtered the bool is set to false.
func (p *page) filterLink(link string) (string, bool) {
	linkURL, err := url.Parse(link)
	if err != nil {
		// TODO I need to consider some debug logging
		return "", false
	}
	if linkURL.Scheme == "" {
		linkURL = p.url.ResolveReference(linkURL)
	}
	if linkURL.Path == "" {
		linkURL.Path = "/"
	}
	if linkURL.Scheme != "http" && linkURL.Scheme != "https" {
		return "", false
	}
	if linkURL.Host != p.url.Host || linkURL.Path == p.url.Path {
		return "", false
	}

	return linkURL.Path, true
}
