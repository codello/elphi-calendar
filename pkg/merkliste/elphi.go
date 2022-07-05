package merkliste

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	ttlcache "github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/codello/elphi-calendar/pkg/metrics"
)

var (
	// ErrInvalidUserID indicates that the requested user ID is formatted
	// incorrectly or does not exist.
	ErrInvalidUserID = errors.New("invalid user id")
	// ErrInvalidEventID indicates that the requested event ID is formatted
	// incorrectly or does not exist.
	ErrInvalidEventID = errors.New("invalid event id")
	// ErrNoEvents indicates that an ics URL did return an empty calendar.
	ErrNoEvents = errors.New("invalid ics file (no events)")
	// ErrMultipleEvents indicates that an ics URL did return multiple events
	// instead of a single one.
	ErrMultipleEvents = errors.New("invalid ics file (multiple events)")
)

// ElphiEvent corresponds to an event as returned by the Elbphilharmonie REST
// API.
type ElphiEvent struct {
	ID               string `json:"evis_id"`
	Title            string `json:"title_de"`
	Subtitle         string `json:"subtitle_de"`
	Description      string `json:"description_long_de"`
	Room             string `json:"room_dispname"`
	ImageURL         string `json:"image_url"`
	ImageCopyright   string `json:"image_copyright_de"`
	WebsiteURL       string `json:"website_url"`
	StartDate        string `json:"date_start"`
	EndDate          string `json:"date_end"`
	ModificationDate string `json:"modified_at"`
	URL              string `json:"url"`
	HTML             string `json:"item_html"`
}

// GetMerkliste returns the merkliste (favorites list) of the user with the
// specified ID. This function returns a list of event IDs.
func GetMerkliste(userID string) ([]string, error) {
	resp, err := http.Get("https://merkliste.elbphilharmonie.de/api/" + userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrInvalidUserID
	}
	var merkliste struct {
		Events map[string]interface{} `json:"events"`
	}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&merkliste)
	if err != nil {
		return nil, err
	}
	events := make([]string, len(merkliste.Events))
	idx := 0
	for eventID := range merkliste.Events {
		events[idx] = eventID
		idx++
	}
	return events, nil
}

// GetElphiEvent fetches the event with the specified ID from the
// Elbphilharmonie API.
//
// For a cached variant of this method see CachedMerkliste.
func GetElphiEvent(eventID string) (*ElphiEvent, error) {
	resp, err := http.Get("https://www.elbphilharmonie.de/de/api/booking/evis/" + url.PathEscape(eventID) + "/")
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrInvalidEventID
	}
	decoder := json.NewDecoder(resp.Body)
	event := &ElphiEvent{}
	err = decoder.Decode(event)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// GetICSEvent fetches the iCal event for the specified event.
//
// For a cached variant of this method see CachedMerkliste.
func GetICSEvent(event *ElphiEvent) (*ics.VEvent, error) {
	resp, err := http.Get(event.WebsiteURL + ".ics")
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	calendar, err := ics.ParseCalendar(resp.Body)
	if err != nil {
		return nil, err
	}
	events := calendar.Events()
	if len(events) == 0 {
		return nil, ErrNoEvents
	}
	if len(events) > 1 {
		return nil, ErrMultipleEvents
	}
	icsEvent := events[0]
	fixupEvent(icsEvent, event)
	return icsEvent, nil
}

// fixupEvent performs some cleanup operations on an event returned by the
// Elbphilharmonie to make for a nicer formatting.
func fixupEvent(icsEvent *ics.VEvent, elphiEvent *ElphiEvent) {
	icsEvent.SetSummary(elphiEvent.Subtitle)
	description := icsEvent.GetProperty(ics.ComponentPropertyDescription).Value
	icsEvent.SetDescription(elphiEvent.Title + "\n\n" + strings.ReplaceAll(description, "\\n", "\n"))
}

// GetCalendar creates an iCal calendar with all events from the merkliste of
// the user with the specified ID.
//
// For a cached variant of this method see CachedMerkliste.
func GetCalendar(userID string, productID string, name string) (*ics.Calendar, error) {
	cal := newCalendar(productID, name)
	merkliste, err := GetMerkliste(userID)
	if err != nil {
		return nil, err
	}
	var elphiEvent *ElphiEvent
	var icsEvent *ics.VEvent
	for _, eventID := range merkliste {
		elphiEvent, err = GetElphiEvent(eventID)
		if err != nil {
			return nil, err
		}
		icsEvent, err = GetICSEvent(elphiEvent)
		if err != nil {
			return nil, err
		}
		cal.AddVEvent(icsEvent)
	}
	return cal, nil
}

// newCalendar is a helper function that creates an iCal calendar with some
// pre-filled config options.
func newCalendar(productID string, name string) *ics.Calendar {
	cal := ics.NewCalendarFor(productID)
	cal.SetMethod(ics.MethodPublish)
	cal.SetCalscale("GREGORIAN")
	cal.SetName(name)
	cal.SetXWRCalName(name)
	return cal
}

// A CachedMerkliste provides a convenient interface to perform cached lookup of
// events on the Elbphilharmonie API.
type CachedMerkliste struct {
	TTL        time.Duration
	EventCache *ttlcache.Cache[string, *ElphiEvent]
	ICSCache   *ttlcache.Cache[string, *ics.VEvent]
	ProductID  string
	Name       string
}

// NewCachedMerkliste creates a new CachedMerkliste with the specified cache TTL.
func NewCachedMerkliste(ttl time.Duration) *CachedMerkliste {
	m := &CachedMerkliste{
		TTL: ttl,
		EventCache: ttlcache.New[string, *ElphiEvent](
			ttlcache.WithTTL[string, *ElphiEvent](ttl),
			ttlcache.WithDisableTouchOnHit[string, *ElphiEvent](),
		),
		ICSCache: ttlcache.New[string, *ics.VEvent](
			ttlcache.WithTTL[string, *ics.VEvent](ttl),
			ttlcache.WithDisableTouchOnHit[string, *ics.VEvent](),
		),
		ProductID: "",
		Name:      "",
	}
	return m
}

// StartCacheExpiration begins the automatic cache expiration.
func (m *CachedMerkliste) StartCacheExpiration() {
	go m.EventCache.Start()
	go m.ICSCache.Start()
}

// StopCacheExpiration stops the automatic cache expiration.
func (m *CachedMerkliste) StopCacheExpiration() {
	m.EventCache.Stop()
	m.ICSCache.Stop()
}

// RegisterMetrics adds caching metrics to the global prometheus registry.
func (m *CachedMerkliste) RegisterMetrics(variableLabels []string, staticLabels prometheus.Labels) {
	eventLabels := prometheus.Labels{}
	icsLabels := prometheus.Labels{}
	for k, v := range staticLabels {
		eventLabels[k] = v
		icsLabels[k] = v
	}
	eventLabels["cache"] = "events"
	icsLabels["cache"] = "ics"
	prometheus.MustRegister(metrics.NewCacheCollector(m.EventCache, "merkliste_", variableLabels, eventLabels))
	prometheus.MustRegister(metrics.NewCacheCollector(m.ICSCache, "merkliste_", variableLabels, icsLabels))
}

// GetElphiEvent performs a cached request for the specified event. Behind the
// scenes this method uses the GetElphiEvent function.
func (m *CachedMerkliste) GetElphiEvent(eventID string) (*ElphiEvent, error) {
	if item := m.EventCache.Get(eventID); item != nil {
		return item.Value(), nil
	}
	event, err := GetElphiEvent(eventID)
	if err != nil {
		return nil, err
	}
	m.EventCache.Set(eventID, event, ttlcache.DefaultTTL)
	return event, nil
}

// GetICSEvent performs a cached request for the specified ICS file. Behind the
// scenes this method uses the GetICSEvent function.
func (m *CachedMerkliste) GetICSEvent(event *ElphiEvent) (*ics.VEvent, error) {
	if item := m.ICSCache.Get(event.ID); item != nil {
		return item.Value(), nil
	}
	icsEvent, err := GetICSEvent(event)
	if err != nil {
		return nil, err
	}
	m.ICSCache.Set(event.ID, icsEvent, ttlcache.DefaultTTL)
	return icsEvent, nil
}

// GetCalendar creates an iCal calendar with the merkliste events of the
// specified user. This works similarly to the GetCalendar function.
func (m *CachedMerkliste) GetCalendar(userID string) (*ics.Calendar, error) {
	cal := m.newCalendar()
	merkliste, err := GetMerkliste(userID)
	if err != nil {
		return nil, err
	}

	// Unfortunately we cannot parallelize this as the Elphi API locks up at too
	// many parallel requests.
	var elphiEvent *ElphiEvent
	var icsEvent *ics.VEvent
	for _, eventID := range merkliste {
		elphiEvent, err = m.GetElphiEvent(eventID)
		if err != nil {
			return nil, err
		}
		icsEvent, err = m.GetICSEvent(elphiEvent)
		if err != nil {
			return nil, err
		}
		cal.AddVEvent(icsEvent)
	}
	return cal, nil
}

// newCalendar creates a new calendar object using the properties of the
// merkliste.
func (m *CachedMerkliste) newCalendar() *ics.Calendar {
	cal := newCalendar(m.ProductID, m.Name)
	cal.SetRefreshInterval("P" + strings.ToUpper(m.TTL.String()))
	return cal
}
