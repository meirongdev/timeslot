// Package integration contains end-to-end tests that start a real HTTP server
// backed by an in-memory SQLite database and exercise the full flow.
package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meirongdev/timeslot/admin"
	"github.com/meirongdev/timeslot/api"
	"github.com/meirongdev/timeslot/config"
	"github.com/meirongdev/timeslot/db"
	"github.com/meirongdev/timeslot/engine"
	"github.com/meirongdev/timeslot/models"
)

// ---------------------------------------------------------------------------
// Test server helpers
// ---------------------------------------------------------------------------

type testServer struct {
srv          *httptest.Server
cfg          *config.Config
calStore     *models.CalendarStore
availStore   *models.AvailabilityStore
busyStore    *models.BusyBlockStore
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	dbFile := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbFile)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	calStore := &models.CalendarStore{DB: database}
	busyStore := &models.BusyBlockStore{DB: database}
	availStore := &models.AvailabilityStore{DB: database}

	cfg := &config.Config{
		AdminUser:       "admin",
		AdminPassword:   "secret",
		SlotDurationMin: 30,
		BufferBeforeMin: 0,
		MaxDaysAhead:    30,
		Timezone:        "UTC",
	}

	eng := &engine.Engine{
		Availability: availStore,
		Busy:         busyStore,
		Location:     time.UTC,
		SlotDuration: time.Duration(cfg.SlotDurationMin) * time.Minute,
		BufferBefore: 0,
	}

	tmplGlob := findTemplates(t)

	mux := http.NewServeMux()

	apiH := &api.Handler{
		Cfg:    cfg,
		Engine: eng,
	}
	apiH.RegisterRoutes(mux)

	adminH := &admin.Handler{
		Cfg:          cfg,
		Calendars:    calStore,
		Availability: availStore,
		SyncWorker:   nil, // sync worker not needed for integration tests
		TemplateDir:  filepath.Dir(tmplGlob),
	}
	adminH.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &testServer{
		srv:        srv,
		cfg:        cfg,
		calStore:   calStore,
		availStore: availStore,
		busyStore:  busyStore,
	}
}

// findTemplates walks up from the working directory to find backend/templates/*.html.
func findTemplates(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "templates", "*.html")
		matches, _ := filepath.Glob(candidate)
		if len(matches) > 0 {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not find templates directory")
	return ""
}

func (ts *testServer) get(t *testing.T, path string) *http.Response {
t.Helper()
resp, err := http.Get(ts.srv.URL + path)
if err != nil {
t.Fatalf("GET %s: %v", path, err)
}
return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
t.Helper()
defer resp.Body.Close()
if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
t.Fatalf("decode JSON: %v", err)
}
}

// seedAvailability adds Monday–Friday 09:00–17:00 UTC rules.
func (ts *testServer) seedAvailability(t *testing.T) {
t.Helper()
for _, wd := range []int{1, 2, 3, 4, 5} { // Mon-Fri
if err := ts.availStore.Create(&models.AvailabilityRule{
Weekday:   wd,
StartTime: "09:00",
EndTime:   "17:00",
}); err != nil {
t.Fatalf("seed availability weekday %d: %v", wd, err)
}
}
}

// nextMonday returns the next Monday at 09:00 UTC.
func nextMonday() time.Time {
now := time.Now().UTC()
daysUntil := (int(time.Monday) - int(now.Weekday()) + 7) % 7
if daysUntil == 0 {
daysUntil = 7
}
d := now.AddDate(0, 0, daysUntil)
return time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// Tests: Slots API
// ---------------------------------------------------------------------------

// TestGetSlots_NoAvailability verifies empty result when no rules are set.
func TestGetSlots_NoAvailability(t *testing.T) {
ts := newTestServer(t)
from := time.Now().UTC()
to := from.Add(7 * 24 * time.Hour)
path := fmt.Sprintf("/api/slots?from=%s&to=%s", from.Format(time.RFC3339), to.Format(time.RFC3339))

resp := ts.get(t, path)
if resp.StatusCode != http.StatusOK {
t.Fatalf("expected 200, got %d", resp.StatusCode)
}
var slots []map[string]any
decodeJSON(t, resp, &slots)
if len(slots) != 0 {
t.Errorf("expected no slots, got %d", len(slots))
}
}

// TestGetSlots_WithAvailability verifies slots are returned when rules exist.
func TestGetSlots_WithAvailability(t *testing.T) {
ts := newTestServer(t)
ts.seedAvailability(t)

monday := nextMonday()
from := monday.Add(-time.Hour)
to := monday.Add(8 * time.Hour)
path := fmt.Sprintf("/api/slots?from=%s&to=%s", from.Format(time.RFC3339), to.Format(time.RFC3339))

resp := ts.get(t, path)
if resp.StatusCode != http.StatusOK {
t.Fatalf("expected 200, got %d", resp.StatusCode)
}
var slots []struct {
Start time.Time `json:"start"`
End   time.Time `json:"end"`
}
decodeJSON(t, resp, &slots)

if len(slots) == 0 {
t.Fatal("expected slots, got none")
}
if !slots[0].Start.Equal(monday) {
t.Errorf("first slot: got %v, want %v", slots[0].Start, monday)
}
for i, s := range slots {
if dur := s.End.Sub(s.Start); dur != 30*time.Minute {
t.Errorf("slot %d: wrong duration %v", i, dur)
}
}
}

// TestGetSlots_DefaultRange verifies the no-params case doesn't error.
func TestGetSlots_DefaultRange(t *testing.T) {
ts := newTestServer(t)
resp := ts.get(t, "/api/slots")
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
t.Fatalf("expected 200, got %d", resp.StatusCode)
}
}

// TestGetSlots_CORSHeaders ensures CORS headers are present.
func TestGetSlots_CORSHeaders(t *testing.T) {
ts := newTestServer(t)
resp := ts.get(t, "/api/slots")
defer resp.Body.Close()
if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
t.Error("missing CORS header")
}
}

// ---------------------------------------------------------------------------
// Tests: Schedule API (privacy-first)
// ---------------------------------------------------------------------------

// TestGetSchedule_NoAvailability verifies empty result when no rules are set.
func TestGetSchedule_NoAvailability(t *testing.T) {
ts := newTestServer(t)
from := time.Now().UTC()
to := from.Add(7 * 24 * time.Hour)
path := fmt.Sprintf("/api/schedule?from=%s&to=%s", from.Format(time.RFC3339), to.Format(time.RFC3339))

resp := ts.get(t, path)
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
t.Fatalf("expected 200, got %d", resp.StatusCode)
}
var entries []map[string]any
decodeJSON(t, resp, &entries)
if len(entries) != 0 {
t.Errorf("expected no entries, got %d", len(entries))
}
}

// TestGetSchedule_AvailableSlots verifies schedule returns available entries.
func TestGetSchedule_AvailableSlots(t *testing.T) {
ts := newTestServer(t)
ts.seedAvailability(t)

monday := nextMonday()
from := monday.Add(-time.Hour)
to := monday.Add(2 * time.Hour)
path := fmt.Sprintf("/api/schedule?from=%s&to=%s", from.Format(time.RFC3339), to.Format(time.RFC3339))

resp := ts.get(t, path)
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
t.Fatalf("expected 200, got %d", resp.StatusCode)
}
var entries []struct {
Start  time.Time `json:"start"`
End    time.Time `json:"end"`
Status string    `json:"status"`
}
decodeJSON(t, resp, &entries)
if len(entries) == 0 {
t.Fatal("expected schedule entries, got none")
}
for _, e := range entries {
if e.Status != "available" {
t.Errorf("expected available, got %s", e.Status)
}
}
}

// TestGetSchedule_OccupiedSlot verifies busy blocks appear as "occupied"
// without revealing event details.
func TestGetSchedule_OccupiedSlot(t *testing.T) {
ts := newTestServer(t)
ts.seedAvailability(t)

monday := nextMonday()
cal := &models.Calendar{Name: "test", Type: "ical_url", URL: "http://example.com", Enabled: true, SyncIntervalMin: 30}
if err := ts.calStore.Create(cal); err != nil {
t.Fatalf("create calendar: %v", err)
}
busyEnd := monday.Add(time.Hour)
if err := ts.busyStore.ReplaceForCalendar(cal.ID, []models.BusyBlock{
{CalendarID: cal.ID, StartAt: monday, EndAt: busyEnd, Title: "SECRET MEETING"},
}); err != nil {
t.Fatalf("seed busy block: %v", err)
}

from := monday.Add(-time.Hour)
to := monday.Add(2 * time.Hour)
path := fmt.Sprintf("/api/schedule?from=%s&to=%s", from.Format(time.RFC3339), to.Format(time.RFC3339))

resp := ts.get(t, path)
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
t.Fatalf("expected 200, got %d", resp.StatusCode)
}

body, _ := io.ReadAll(resp.Body)
if strings.Contains(string(body), "SECRET MEETING") {
t.Error("schedule leaks private event title")
}

var entries []struct {
Status string `json:"status"`
}
if err := json.Unmarshal(body, &entries); err != nil {
t.Fatalf("decode schedule: %v", err)
}
hasOccupied := false
for _, e := range entries {
if e.Status == "occupied" {
hasOccupied = true
}
}
if !hasOccupied {
t.Error("expected at least one occupied entry")
}
}

// TestGetSchedule_CORSHeaders verifies schedule endpoint has CORS headers.
func TestGetSchedule_CORSHeaders(t *testing.T) {
ts := newTestServer(t)
resp := ts.get(t, "/api/schedule")
defer resp.Body.Close()
if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
t.Error("missing CORS header on /api/schedule")
}
}

// ---------------------------------------------------------------------------
// Tests: Admin UI
// ---------------------------------------------------------------------------

// TestAdminAuth_Unauthorized verifies admin routes require Basic Auth.
func TestAdminAuth_Unauthorized(t *testing.T) {
ts := newTestServer(t)
resp, err := http.Get(ts.srv.URL + "/admin/")
if err != nil {
t.Fatalf("GET /admin/: %v", err)
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusUnauthorized {
t.Fatalf("expected 401, got %d", resp.StatusCode)
}
}
