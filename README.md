# SiteMapper
[![Build Status](https://travis-ci.org/tkuhlman/sitemapper.svg)](https://travis-ci.org/tkuhlman/sitemapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/tkuhlman/sitemapper)](https://goreportcard.com/report/github.com/tkuhlman/sitemapper)
[![Coverage Status](https://coveralls.io/repos/github/tkuhlman/sitemapper/badge.svg?branch=master)](https://coveralls.io/github/tkuhlman/sitemapper?branch=master)

A tool for creating a site map from a starting URL.

Starting the mapper is done by simply running the binary with a starting URL and optional flag to specify the number of workers.
Running the Docker container is the easiest `docker run --rm -p 8080:8080 tkuhlman/sitemapper mysite.com`.

When finished a local website displaying the results will be started up.
The site map itself is a simple directed graph which can be downloaded as a JSON file or displayed by the embedded web server.

## Building

All changes are built and tested using [Travis CI](https://travis-ci.org/), see the build status icon.
To build manually download the source and use [dep](https://github.com/golang/dep), for example:

    git clone https://github.com/tkuhlman/sitemapper
    cd sitemapper
    dep ensure
    go build -o sitemapper main.go

Tests can be run with `go test` as is standard for Golang.

## Limitations
- Only html pages are parsed for links and from these only anchor links are retreived so no links from forms, javascript, etc.
- Any non 2XX status code is considered a failure, even redirects.
- URL parsing is not forgiving of simple errors, '/site/', '/site' and '//site' are all different paths.
  Most web servers redirect these slash mistakes this considers redirection an error.

## Wishlist
- Wrap updates to the SiteMap Pages in a sync.RWMutex or use sync.Map so the progress of the map as it builds can be watched from the embedded website.
- Resume after pause.
- Persistng the current progress and resuming from persisted data.
- Rework to use neo4j or similar DB rather than an internal data structure.
  An alternative is to put some more work into optimizing the internal data structure.
- The ability to map multiple sites.
- Intelligent updating of existing data to account for site changes.
- An overall timeout for the entire site crawl, this can be done with a ticker in a go routine that sends
  a shutdown signal or via adding to the select in sm.Start
