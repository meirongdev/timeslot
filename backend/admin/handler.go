// Package admin implements the Admin UI (server-side HTML, basic auth protected).
package admin

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/meirongdev/timeslot/config"
	"github.com/meirongdev/timeslot/models"
	"github.com/meirongdev/timeslot/sync"
)

// Handler holds dependencies for Admin UI.
type Handler struct {
	Cfg          *config.Config
	Calendars    *models.CalendarStore
	Availability *models.AvailabilityStore
	BusyBlocks   *models.BusyBlockStore
	SyncWorker   *sync.Worker
	TemplateDir  string // Path to templates directory, defaults to "templates"
}

// RegisterRoutes attaches all admin routes (all protected by basic auth).
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin/", h.auth(h.route))
}

func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin")
	switch {
	case path == "" || path == "/":
		h.indexPage(w, r)
	case path == "/calendars" || path == "/calendars/":
		h.calendarsPage(w, r)
	case strings.HasPrefix(path, "/calendars/sync/"):
		h.syncCalendar(w, r, path)
	case strings.HasPrefix(path, "/calendars/delete/"):
		h.deleteCalendar(w, r, path)
	case path == "/availability" || path == "/availability/":
		h.availabilityPage(w, r)
	case strings.HasPrefix(path, "/availability/delete/"):
		h.deleteAvailability(w, r, path)
	default:
		http.NotFound(w, r)
	}
}

// ---- index ------------------------------------------------------------------

func (h *Handler) indexPage(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	if h.Cfg.Timezone != "" {
		if loc, err := time.LoadLocation(h.Cfg.Timezone); err == nil {
			now = time.Now().In(loc)
		}
	}

	// First day of current month
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// Last day of current month
	last := first.AddDate(0, 1, -1)

	// Start of grid (Sunday of the week containing 'first')
	start := first.AddDate(0, 0, -int(first.Weekday()))
	// End of grid (Saturday of the week containing 'last')
	end := last.AddDate(0, 0, 6-int(last.Weekday()))

	events, err := h.BusyBlocks.ListRange(start, end.Add(24*time.Hour))
	if err != nil {
		log.Printf("admin: list busy blocks: %v", err)
	}

	cals, _ := h.Calendars.List()
	calMap := make(map[int64]string)
	for _, c := range cals {
		calMap[c.ID] = c.Name
	}

	// Group events by date
	eventRows := make(map[string][]models.BusyBlock)
	for _, e := range events {
		d := e.StartAt.In(now.Location()).Format("2006-01-02")
		eventRows[d] = append(eventRows[d], e)
	}

	// Generate grid days
	type dayInfo struct {
		Date    time.Time
		ISO     string
		Num     int
		IsToday bool
		InMonth bool
		Events  []models.BusyBlock
	}
	var grid []dayInfo
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		iso := d.Format("2006-01-02")
		grid = append(grid, dayInfo{
			Date:    d,
			ISO:     iso,
			Num:     d.Day(),
			IsToday: iso == now.Format("2006-01-02"),
			InMonth: d.Month() == now.Month(),
			Events:  eventRows[iso],
		})
	}

	h.render(w, "index.html", map[string]any{
		"Title":    "Dashboard",
		"Current":  now.Format("January 2006"),
		"Grid":     grid,
		"Calendar": calMap,
	})
}

// ---- calendars --------------------------------------------------------------

func (h *Handler) calendarsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		iv, _ := strconv.Atoi(r.FormValue("sync_interval_min"))
		if iv <= 0 {
			iv = 30
		}
		c := &models.Calendar{
			Name:            r.FormValue("name"),
			Type:            r.FormValue("type"),
			URL:             r.FormValue("url"),
			Credentials:     r.FormValue("credentials"),
			SyncIntervalMin: iv,
			Enabled:         r.FormValue("enabled") == "on",
		}
		if err := h.Calendars.Create(c); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/calendars", http.StatusSeeOther)
		return
	}

	cals, err := h.Calendars.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.render(w, "calendars.html", map[string]any{
		"Title":     "Calendars",
		"Calendars": cals,
		"Error":     r.URL.Query().Get("error"),
		"Success":   r.URL.Query().Get("success") == "1",
	})
}

func (h *Handler) syncCalendar(w http.ResponseWriter, r *http.Request, path string) {
	idStr := strings.TrimPrefix(path, "/calendars/sync/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	c, err := h.Calendars.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.SyncWorker.SyncOne(*c); err != nil {
		log.Printf("admin: force sync calendar %d: %v", id, err)
		http.Redirect(w, r, "/admin/calendars?error="+err.Error(), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/calendars?success=1", http.StatusSeeOther)
}

func (h *Handler) deleteCalendar(w http.ResponseWriter, r *http.Request, path string) {
	idStr := strings.TrimPrefix(path, "/calendars/delete/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	h.Calendars.Delete(id)
	http.Redirect(w, r, "/admin/calendars", http.StatusSeeOther)
}

// ---- availability -----------------------------------------------------------

func (h *Handler) availabilityPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		weekday, _ := strconv.Atoi(r.FormValue("weekday"))
		rule := &models.AvailabilityRule{
			Weekday:   weekday,
			StartTime: r.FormValue("start_time"),
			EndTime:   r.FormValue("end_time"),
		}
		if vf := r.FormValue("valid_from"); vf != "" {
			t, err := time.Parse("2006-01-02", vf)
			if err == nil {
				rule.ValidFrom = &t
			}
		}
		if vu := r.FormValue("valid_until"); vu != "" {
			t, err := time.Parse("2006-01-02", vu)
			if err == nil {
				rule.ValidUntil = &t
			}
		}
		if err := h.Availability.Create(rule); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/availability", http.StatusSeeOther)
		return
	}

	rules, err := h.Availability.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	weekdays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	h.render(w, "availability.html", map[string]any{
		"Title":    "Availability Rules",
		"Rules":    rules,
		"Weekdays": weekdays,
	})
}

func (h *Handler) deleteAvailability(w http.ResponseWriter, r *http.Request, path string) {
	idStr := strings.TrimPrefix(path, "/availability/delete/")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	h.Availability.Delete(id)
	http.Redirect(w, r, "/admin/availability", http.StatusSeeOther)
}

// ---- helpers ----------------------------------------------------------------

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	dir := h.TemplateDir
	if dir == "" {
		dir = "templates"
	}
	tmpl, err := template.ParseFiles(dir+"/layout.html", dir+"/"+name)
	if err != nil {
		log.Printf("admin: template %s parse (dir=%s): %v", name, dir, err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("admin: template %s execute: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != h.Cfg.AdminUser || pass != h.Cfg.AdminPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="timeslot admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
