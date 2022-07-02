package merkliste

import (
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/codello/elphi-calendar/pkg/metrics"
)

// The ErrorLogger is used to log errors.
var ErrorLogger = log.New(os.Stdout, "", log.LstdFlags)

// Handler provides an HTTP endpoint for the merkliste calendar. This implements
// http.Handler.
type Handler struct {
	Merkliste *CachedMerkliste
}

// RegisterMetrics adds caching metrics to the global prometheus registry.
func (h *Handler) RegisterMetrics() {
	prometheus.MustRegister(metrics.NewCacheCollector(
		h.Merkliste.EventCache, nil, prometheus.Labels{
			"cache": "events",
		},
	))
	log.Println()
	prometheus.MustRegister(metrics.NewCacheCollector(
		h.Merkliste.ICSCache, nil, prometheus.Labels{
			"cache": "ics",
		},
	))
}

// ServeHTTP implements an iCal HTTP endpoint providing events from a merkliste.
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	components := strings.Split(req.URL.Path, "/")
	userID := strings.TrimSuffix(components[len(components)-1], ".ics")
	cal, err := h.Merkliste.GetCalendar(userID)
	if err != nil && errors.Is(err, ErrInvalidUserID) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Invalid User ID: "+userID)
		return
	}
	if err != nil {
		ErrorLogger.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, http.StatusText(http.StatusServiceUnavailable))
	}
	w.Header().Set("Content-Type", "text/calendar")
	cal.SerializeTo(w)
}
