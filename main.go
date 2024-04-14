package main

import (
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"

	"codello.dev/elphi-calendar/elphi"
	"codello.dev/elphi-calendar/metrics"
)

var (
	bindAddress string
	cacheTTL    time.Duration
	certFile    string
	keyFile     string
	creator     string
	name        string
)

var version string

func init() {
	cmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
	cmd.Flags().StringVarP(&bindAddress, "bind-address", "a", ":8080", "The address on which the server listens.")
	cmd.Flags().DurationVar(&cacheTTL, "ttl", 1*time.Hour, "The amount of time after which cached events expire and need to be re-fetched.")
	cmd.Flags().StringVar(&certFile, "cert-file", "", "Path to the TLS server certificate.")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "Path to the TLS private key.")
	cmd.Flags().StringVar(&creator, "creator", "elphi-calendar", "The value of the creator field in generated ICS files.")
	cmd.Flags().StringVarP(&name, "name", "n", "Elbphilharmonie Merkliste", "The suggested name for the calendar.")

	if info, ok := debug.ReadBuildInfo(); version == "" && ok {
		var commit string
		var dirty bool
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				commit = setting.Value
			case "vcs.modified":
				dirty = setting.Value == "true"
			}
		}
		if commit != "" {
			version = commit
			if dirty {
				version += ".dirty"
			}
		}
		if version == "" {
			version = "unknown"
		}
		cmd.Version = version
	}
}

var cmd = &cobra.Command{
	Use:               "elphi-calendar",
	Short:             "elphi-calendar is a calendar server for your Elbphilharmonie favorites.",
	Long:              "Subscribe to your favorite events via an ICS URL.",
	Version:           version,
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Configure the App
		log.SetPrefix("[INFO] ")
		elphi.ErrorLogger.SetPrefix("[ERROR] ")
		cache := elphi.NewCachedMerkliste(cacheTTL)
		cache.Name = name
		cache.ProductID = creator
		prometheus.MustRegister(metrics.NewCacheCollector(cache.EventCache, "merkliste_", nil, prometheus.Labels{"cache": "events"}))
		prometheus.MustRegister(metrics.NewCacheCollector(cache.ICSCache, "merkliste_", nil, prometheus.Labels{"cache": "ics"}))
		cache.StartCacheExpiration()

		// Setup router
		http.Handle("GET /merkliste/{userID}", &elphi.CalendarHandler{Merkliste: cache})
		http.Handle("GET /events/{id}", &elphi.EventHandler{Merkliste: cache})
		http.Handle("GET /metrics", promhttp.Handler())
		http.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		if certFile != "" && keyFile != "" {
			log.Println("Running on " + bindAddress + " (TLS on)")
			return http.ListenAndServeTLS(bindAddress, certFile, keyFile, nil)
		} else {
			log.Println("Running on " + bindAddress + " (TLS off)")
			return http.ListenAndServe(bindAddress, nil)
		}
	},
}

// main is the main entrypoint of the program.
func main() {
	if cmd.Execute() != nil {
		os.Exit(1)
	}
}
