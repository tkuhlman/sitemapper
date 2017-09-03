package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/tkuhlman/sitemapper/mapper"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	workers       = flag.Uint("w", 4, "The number of worker go routines connecting to sites simultaneously")
	listenAddress = flag.String("l", "0.0.0.0:8080", "The listen address and port for the embedded webserver")
)

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("The URL to begin the site mapping from is required and the only valid non-flag argument.")
		// TODO better help and usage output
	}
	sm, err := mapper.NewSiteMap(flag.Arg(0), *workers)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/metrics", prometheus.UninstrumentedHandler())
	go func() {
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
	}()

	log.Printf("Crawling site %s", sm.URL)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go sm.HandleSignals(signals)

	if err := sm.Start(); err != nil {
		log.Printf("Site crawling unfinished: %v", err)
	}

	// TODO determine the primary ip when listenAddress = 0.0.0.0
	ip := "localhost"
	log.Printf("The sitemap results are available at http://%s/", ip)

	// TODO wait for the web site to exit or to receive a SIGINT, see sitemap.shutdown channel
	fmt.Printf("Site %s - %d pages\n", sm.URL, len(sm.Pages))
	for path, p := range sm.Pages {
		if p.Broken {
			fmt.Printf("\t%s -> ! Broken\n", path)
			continue
		}
		for subPath := range p.Links {
			fmt.Printf("\t%s -> %s\n", path, subPath)
		}
	}
}
