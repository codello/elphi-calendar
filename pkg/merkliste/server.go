package merkliste

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
)

// The ErrorLogger is used to log errors.
var ErrorLogger = log.New(os.Stdout, "", log.LstdFlags)

// CalendarHandler provides an HTTP endpoint for the merkliste calendar. This
// implements http.Handler.
type CalendarHandler struct {
	Prefix    string
	Merkliste *CachedMerkliste
}

// ServeHTTP implements an iCal HTTP endpoint providing events from a merkliste.
func (h *CalendarHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	components := strings.Split(req.URL.Path, "/")
	userID := strings.TrimSuffix(components[len(components)-1], ".ics")
	cal, err := h.Merkliste.GetCalendar(userID)
	if errors.Is(err, ErrInvalidUserID) {
		http.Error(w, "Invalid User ID: "+userID, http.StatusNotFound)
		return
	}
	if err != nil {
		ErrorLogger.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/calendar")
	cal.SerializeTo(w)
}

// EventHandler provides an HTTP endpoint for a calendar containing a single
// event. This implements http.Handler.
type EventHandler struct {
	Prefix    string
	Merkliste *CachedMerkliste
}

// ServeHTTP implements an iCal HTTP endpoint providing single Elphi events.
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	eventID := strings.TrimPrefix(req.URL.Path, h.Prefix)
	elphiEvent, err := h.Merkliste.GetElphiEvent(eventID)
	if errors.Is(err, ErrInvalidEventID) {
		http.Error(w, "Invalid Event ID: "+eventID, http.StatusNotFound)
		return
	}
	if err != nil {
		ErrorLogger.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	icsEvent, err := h.Merkliste.GetICSEvent(elphiEvent)
	if err != nil {
		ErrorLogger.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	cal := h.Merkliste.newCalendar()
	cal.AddVEvent(icsEvent)
	w.Header().Set("Content-Type", "text/calendar")
	cal.SerializeTo(w)
}
