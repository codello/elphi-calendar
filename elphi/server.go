package elphi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
)

// The ErrorLogger is used to log errors.
var ErrorLogger = log.New(os.Stdout, "", log.LstdFlags)

// CalendarHandler provides an HTTP endpoint for the elphi calendar. This
// implements http.Handler.
type CalendarHandler struct {
	Merkliste *CachedMerkliste
}

// ServeHTTP implements an iCal HTTP endpoint providing events from a elphi.
func (h *CalendarHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	userID := strings.TrimSuffix(req.PathValue("userID"), ".ics")
	cal, err := h.Merkliste.GetCalendar(ctx, userID)
	if errors.Is(err, context.Canceled) {
		return
	}
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
	Merkliste *CachedMerkliste
}

// ServeHTTP implements an iCal HTTP endpoint providing single Elphi events.
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	eventID := req.PathValue("id")
	elphiEvent, err := h.Merkliste.GetEvent(ctx, eventID)
	if errors.Is(err, context.Canceled) {
		return
	}
	if errors.Is(err, ErrInvalidEventID) {
		http.Error(w, "Invalid Event ID: "+eventID, http.StatusNotFound)
		return
	}
	if err != nil {
		ErrorLogger.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	icsEvent, err := h.Merkliste.GetICSEvent(ctx, elphiEvent)
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		ErrorLogger.Println(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	cal := newCalendar(h.Merkliste.ProductID, h.Merkliste.Name)
	cal.SetRefreshInterval("P" + strings.ToUpper(h.Merkliste.TTL.String()))
	cal.AddVEvent(icsEvent)
	w.Header().Set("Content-Type", "text/calendar")
	cal.SerializeTo(w)
}
