// Package sync fetches remote iCal feeds and stores busy blocks.
package sync

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	ical "github.com/arran4/golang-ical"

	"github.com/meirongdev/timeslot/models"
)

// Worker pulls iCal data for all enabled calendars and writes busy blocks.
type Worker struct {
	Calendars  *models.CalendarStore
	BusyBlocks *models.BusyBlockStore
	HTTPClient *http.Client
}

// SyncAll fetches every enabled calendar.
func (w *Worker) SyncAll() {
	cals, err := w.Calendars.List()
	if err != nil {
		log.Printf("sync: list calendars: %v", err)
		return
	}
	for _, c := range cals {
		if !c.Enabled {
			continue
		}
		if err := w.SyncOne(c); err != nil {
			log.Printf("sync: calendar %d (%s): %v", c.ID, c.Name, err)
		}
	}
}

// SyncOne fetches and processes a single calendar.
func (w *Worker) SyncOne(c models.Calendar) error {
	blocks, err := w.fetchIcal(c.URL)
	if err != nil {
		return fmt.Errorf("fetch ical: %w", err)
	}
	if err := w.BusyBlocks.ReplaceForCalendar(c.ID, blocks); err != nil {
		return fmt.Errorf("store blocks: %w", err)
	}
	return w.Calendars.TouchSynced(c.ID)
}

func (w *Worker) fetchIcal(rawURL string) ([]models.BusyBlock, error) {
	client := w.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Timeslot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return parseIcal(resp.Body)
}

func parseIcal(r io.Reader) ([]models.BusyBlock, error) {
	cal, err := ical.ParseCalendar(r)
	if err != nil {
		return nil, fmt.Errorf("parse ical: %w", err)
	}

	var blocks []models.BusyBlock
	for _, comp := range cal.Events() {
		dtStart := comp.GetProperty(ical.ComponentPropertyDtStart)
		dtEnd := comp.GetProperty(ical.ComponentPropertyDtEnd)
		if dtStart == nil || dtEnd == nil {
			continue
		}
		start, err := parseICalTime(dtStart.Value)
		if err != nil {
			continue
		}
		end, err := parseICalTime(dtEnd.Value)
		if err != nil {
			continue
		}

		var title string
		if p := comp.GetProperty(ical.ComponentPropertySummary); p != nil {
			title = p.Value
		}

		blocks = append(blocks, models.BusyBlock{
			StartAt: start,
			EndAt:   end,
			Title:   title,
		})
	}
	return blocks, nil
}

func parseICalTime(val string) (time.Time, error) {
	// Try RFC3339 first
	t, err := time.Parse(time.RFC3339, val)
	if err == nil {
		return t, nil
	}
	// Try iCal date-time format: 20060102T150405Z
	t, err = time.Parse("20060102T150405Z", val)
	if err == nil {
		return t, nil
	}
	// Try iCal date format: 20060102
	t, err = time.Parse("20060102", val)
	if err == nil {
		return t, nil
	}
	return time.Time{}, err
}
