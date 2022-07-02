package main

import (
	"fmt"
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
	cache.StartCacheExpiration()
	handler := &merkliste.Handler{Merkliste: cache}
	handler.RegisterMetrics()

	// Setup router
	http.Handle("/merkliste/", handler)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "OK")
	})

	if options.CertFile != "" && options.KeyFile != "" {
		log.Println("Running on " + options.BindAddress + " (TLS on)")
		log.Fatal(http.ListenAndServeTLS(options.BindAddress, options.CertFile, options.KeyFile, nil))
	} else {
		log.Println("Running on " + options.BindAddress + " (TLS off)")
		log.Fatal(http.ListenAndServe(options.BindAddress, nil))
	}
}
