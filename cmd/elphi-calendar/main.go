package main

import (
	"log"
	"net/http"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/codello/elphi-calendar/pkg/merkliste"
)

// main is the main entrypoint of the program.
func main() {
	// Parse CLI arguments
	_, err := flags.Parse(&options)
	if err != nil {
		if flags.WroteHelp(err) {
			os.Exit(0)
		}
		os.Exit(1)
	}

	// Configure the App
	log.SetPrefix("[INFO] ")
	merkliste.ErrorLogger.SetPrefix("[ERROR] ")
	cache := merkliste.NewCachedMerkliste(options.CacheTTL)
	cache.Name = options.Name
	cache.ProductID = options.Creator
	cache.RegisterMetrics(nil, nil)
	cache.StartCacheExpiration()

	// Setup router
	http.Handle("/merkliste/", &merkliste.CalendarHandler{Merkliste: cache, Prefix: "/merkliste/"})
	http.Handle("/events/", &merkliste.EventHandler{Merkliste: cache, Prefix: "/events/"})
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	if options.CertFile != "" && options.KeyFile != "" {
		log.Println("Running on " + options.BindAddress + " (TLS on)")
		log.Fatal(http.ListenAndServeTLS(options.BindAddress, options.CertFile, options.KeyFile, nil))
	} else {
		log.Println("Running on " + options.BindAddress + " (TLS off)")
		log.Fatal(http.ListenAndServe(options.BindAddress, nil))
	}
}
