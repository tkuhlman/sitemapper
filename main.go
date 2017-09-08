package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	crawlSignals := make(chan os.Signal, 2)
	signal.Notify(crawlSignals, syscall.SIGINT, syscall.SIGTERM)
	go sm.HandleSignals(crawlSignals)

	if err := sm.Start(); err != nil {
		log.Printf("Site crawling unfinished: %v", err)
	}

	resultSignals := make(chan os.Signal, 2)
	signal.Notify(resultSignals, syscall.SIGINT, syscall.SIGTERM)

	http.Handle("/", http.FileServer(http.Dir("./webroot/")))
	http.Handle("/json", sm)
	listenSplit := strings.SplitN(*listenAddress, ":", 2)
	ip := listenSplit[0]
	if listenSplit[0] == "0.0.0.0" {
		ip = "localhost"
	}
	log.Printf("The sitemap results are available at http://%s:%s/", ip, listenSplit[1])
	log.Print("Ctrl-C will stop the results webserver and exit.")

	<-resultSignals
}
