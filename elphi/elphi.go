package elphi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/jellydator/ttlcache/v3"
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

// Event corresponds to an event as returned by the Elbphilharmonie REST
// API.
type Event struct {
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

// GetMerkliste returns the elphi (favorites list) of the user with the
// specified ID. This function returns a list of event IDs.
func GetMerkliste(ctx context.Context, userID string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://merkliste.elbphilharmonie.de/api/"+userID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
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

// GetEvent fetches the event with the specified ID from the
// Elbphilharmonie API.
//
// For a cached variant of this method see CachedMerkliste.
func GetEvent(ctx context.Context, eventID string) (*Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.elbphilharmonie.de/de/api/booking/evis/"+url.PathEscape(eventID)+"/", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrInvalidEventID
	}
	decoder := json.NewDecoder(resp.Body)
	event := &Event{}
	err = decoder.Decode(event)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// GetICSEvent fetches the iCal event for the specified event.
//
// For a cached variant of this method see CachedMerkliste.
func GetICSEvent(ctx context.Context, event *Event) (*ics.VEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, event.WebsiteURL+".ics", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
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
func fixupEvent(icsEvent *ics.VEvent, elphiEvent *Event) {
	id := icsEvent.GetProperty(ics.ComponentPropertyUniqueId).Value
	icsEvent.SetProperty(ics.ComponentPropertyUniqueId, "custom-"+id)
	icsEvent.SetSummary(elphiEvent.Subtitle)
	description := icsEvent.GetProperty(ics.ComponentPropertyDescription).Value
	icsEvent.SetDescription(elphiEvent.Title + "\n\n" + strings.ReplaceAll(description, "\\n", "\n"))
}

// GetCalendar creates an iCal calendar with all events from the merkliste of
// the user with the specified ID.
//
// For a cached variant of this method see CachedMerkliste.
func GetCalendar(ctx context.Context, userID string, productID string, name string) (*ics.Calendar, error) {
	cal := newCalendar(productID, name)
	merkliste, err := GetMerkliste(ctx, userID)
	if err != nil {
		return nil, err
	}
	var elphiEvent *Event
	var icsEvent *ics.VEvent
	for _, eventID := range merkliste {
		elphiEvent, err = GetEvent(ctx, eventID)
		if err != nil {
			return nil, err
		}
		icsEvent, err = GetICSEvent(ctx, elphiEvent)
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
	EventCache *ttlcache.Cache[string, *Event]
	ICSCache   *ttlcache.Cache[string, *ics.VEvent]
	ProductID  string
	Name       string
}

// NewCachedMerkliste creates a new CachedMerkliste with the specified cache TTL.
func NewCachedMerkliste(ttl time.Duration) *CachedMerkliste {
	return &CachedMerkliste{
		TTL: ttl,
		EventCache: ttlcache.New[string, *Event](
			ttlcache.WithTTL[string, *Event](ttl),
			ttlcache.WithDisableTouchOnHit[string, *Event](),
		),
		ICSCache: ttlcache.New[string, *ics.VEvent](
			ttlcache.WithTTL[string, *ics.VEvent](ttl),
			ttlcache.WithDisableTouchOnHit[string, *ics.VEvent](),
		),
		ProductID: "",
		Name:      "",
	}
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

// GetEvent performs a cached request for the specified event. Behind the
// scenes this method uses the GetEvent function.
func (m *CachedMerkliste) GetEvent(ctx context.Context, eventID string) (*Event, error) {
	if item := m.EventCache.Get(eventID); item != nil {
		return item.Value(), nil
	}
	event, err := GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	m.EventCache.Set(eventID, event, ttlcache.DefaultTTL)
	return event, nil
}

// GetICSEvent performs a cached request for the specified ICS file. Behind the
// scenes this method uses the GetICSEvent function.
func (m *CachedMerkliste) GetICSEvent(ctx context.Context, event *Event) (*ics.VEvent, error) {
	if item := m.ICSCache.Get(event.ID); item != nil {
		return item.Value(), nil
	}
	icsEvent, err := GetICSEvent(ctx, event)
	if err != nil {
		return nil, err
	}
	m.ICSCache.Set(event.ID, icsEvent, ttlcache.DefaultTTL)
	return icsEvent, nil
}

// GetCalendar creates an iCal calendar with the elphi events of the
// specified user. This works similarly to the GetCalendar function.
func (m *CachedMerkliste) GetCalendar(ctx context.Context, userID string) (*ics.Calendar, error) {
	merkliste, err := GetMerkliste(ctx, userID)
	if err != nil {
		return nil, err
	}

	cal := newCalendar(m.ProductID, m.Name)
	cal.SetRefreshInterval("P" + strings.ToUpper(m.TTL.String()))
	// Unfortunately we cannot parallelize this as the Elphi API locks up at too
	// many parallel requests.
	var elphiEvent *Event
	var icsEvent *ics.VEvent
	for _, eventID := range merkliste {
		elphiEvent, err = m.GetEvent(ctx, eventID)
		if err != nil {
			return nil, err
		}
		icsEvent, err = m.GetICSEvent(ctx, elphiEvent)
		if err != nil {
			return nil, err
		}
		cal.AddVEvent(icsEvent)
	}
	return cal, nil
}
