package main

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"

	"codello.dev/elphi-calendar/pkg/merkliste"
)

var cmd = &cobra.Command{
	Use:               "elphi-calendar",
	Short:             "elphi-calendar is a calendar server for your Elbphilharmonie favorites.",
	Long:              "Subscribe to your favorite events via an ICS URL.",
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	RunE:              run,
}

var (
	bindAddress string
	cacheTTL    time.Duration
	certFile    string
	keyFile     string
	creator     string
	name        string

	version bool
)

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
	cmd.Flags().BoolVarP(&version, "version", "v", false, "Show the version of the program and exit.")
}

// main is the main entrypoint of the program.
func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}

// run is the main entrypoint of the program after arguments have been parsed.
func run(cmd *cobra.Command, args []string) error {
	if version {
		printVersion()
		return nil
	}

	// Configure the App
	log.SetPrefix("[INFO] ")
	merkliste.ErrorLogger.SetPrefix("[ERROR] ")
	cache := merkliste.NewCachedMerkliste(cacheTTL)
	cache.Name = name
	cache.ProductID = creator
	cache.RegisterMetrics(nil, nil)
	cache.StartCacheExpiration()

	// Setup router
	http.Handle("/merkliste/", &merkliste.CalendarHandler{Merkliste: cache, Prefix: "/merkliste/"})
	http.Handle("/events/", &merkliste.EventHandler{Merkliste: cache, Prefix: "/events/"})
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	if certFile != "" && keyFile != "" {
		log.Println("Running on " + bindAddress + " (TLS on)")
		return http.ListenAndServeTLS(bindAddress, certFile, keyFile, nil)
	} else {
		log.Println("Running on " + bindAddress + " (TLS off)")
		return http.ListenAndServe(bindAddress, nil)
	}
}
