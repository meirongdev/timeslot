// Package engine computes bookable time slots.
//
// Algorithm:
//  1. Expand availability_rules into concrete (start, end) windows for the
//     requested date range.
//  2. Subtract busy_blocks to get free windows.
//  3. Slice each free window into slots of slotDuration.
//  4. Drop slots whose start is within bufferBefore of now.
package engine

import (
	"fmt"
	"sort"
	"time"

	"github.com/meirongdev/timeslot/models"
)

// Slot is a bookable time slot.
type Slot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ScheduleEntry is a single entry in the privacy-first schedule view.
// Status is either "available" (bookable) or "occupied" (busy, details hidden).
type ScheduleEntry struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Status string    `json:"status"` // "available" | "occupied"
}

// Engine holds the configuration needed for slot calculation.
type Engine struct {
	Availability *models.AvailabilityStore
	Busy         *models.BusyBlockStore
	Location     *time.Location
	SlotDuration time.Duration
	BufferBefore time.Duration
}

// ComputeSlots returns all available slots in [from, to).
func (e *Engine) ComputeSlots(from, to time.Time) ([]Slot, error) {
	rules, err := e.Availability.List()
	if err != nil {
		return nil, fmt.Errorf("engine: list availability: %w", err)
	}

	blocks, err := e.Busy.ListRange(from, to)
	if err != nil {
		return nil, fmt.Errorf("engine: list busy: %w", err)
	}

	// Build a flat list of "unavailable" intervals.
	var occupied []interval
	for _, b := range blocks {
		occupied = append(occupied, interval{b.StartAt, b.EndAt})
	}
	sortIntervals(occupied)

	// Expand rules into concrete windows.
	windows := e.expandRules(rules, from, to)

	// Subtract occupied from windows.
	var freeWindows []interval
	for _, w := range windows {
		freeWindows = append(freeWindows, subtract(w, occupied)...)
	}

	// Slice into fixed-duration slots.
	now := time.Now()
	cutoff := now.Add(e.BufferBefore)
	var slots []Slot
	for _, fw := range freeWindows {
		start := fw.start
		for {
			end := start.Add(e.SlotDuration)
			if end.After(fw.end) {
				break
			}
			if start.After(cutoff) || start.Equal(cutoff) {
				slots = append(slots, Slot{Start: start, End: end})
			}
			start = end
		}
	}
	return slots, nil
}

// ComputeSchedule returns a privacy-first schedule view for [from, to).
// Each entry is "available" (bookable slot) or "occupied" (busy, details hidden).
// Only times within availability windows are included.
func (e *Engine) ComputeSchedule(from, to time.Time) ([]ScheduleEntry, error) {
	rules, err := e.Availability.List()
	if err != nil {
		return nil, fmt.Errorf("engine: list availability: %w", err)
	}

	blocks, err := e.Busy.ListRange(from, to)
	if err != nil {
		return nil, fmt.Errorf("engine: list busy: %w", err)
	}

	// Build flat occupied list.
	var occupied []interval
	for _, b := range blocks {
		occupied = append(occupied, interval{b.StartAt, b.EndAt})
	}
	sortIntervals(occupied)
	occupied = mergeIntervals(occupied)

	// Expand availability windows.
	windows := e.expandRules(rules, from, to)

	now := time.Now()
	cutoff := now.Add(e.BufferBefore)

	var entries []ScheduleEntry
	for _, w := range windows {
		// Find occupied sub-intervals within this window.
		freeWindows := subtract(w, occupied)

		// Mark occupied gaps as "occupied".
		// Walk through free windows and infer gaps as occupied.
		cursor := w.start
		for _, fw := range freeWindows {
			if fw.start.After(cursor) {
				// Gap between cursor and fw.start is occupied.
				entries = append(entries, ScheduleEntry{
					Start:  cursor,
					End:    fw.start,
					Status: "occupied",
				})
			}
			// Slice free window into slots.
			slotStart := fw.start
			for {
				slotEnd := slotStart.Add(e.SlotDuration)
				if slotEnd.After(fw.end) {
					break
				}
				if slotStart.After(cutoff) || slotStart.Equal(cutoff) {
					entries = append(entries, ScheduleEntry{
						Start:  slotStart,
						End:    slotEnd,
						Status: "available",
					})
				}
				slotStart = slotEnd
			}
			cursor = fw.end
		}
		// Remaining portion of window after last free segment is occupied.
		if cursor.Before(w.end) {
			entries = append(entries, ScheduleEntry{
				Start:  cursor,
				End:    w.end,
				Status: "occupied",
			})
		}
	}
	return entries, nil
}



type interval struct {
	start, end time.Time
}

func (e *Engine) expandRules(rules []models.AvailabilityRule, from, to time.Time) []interval {
	var out []interval
	// Iterate day by day.
	loc := e.Location
	if loc == nil {
		loc = time.UTC
	}
	day := truncateToDay(from.In(loc))
	end := truncateToDay(to.In(loc)).Add(24 * time.Hour)
	for day.Before(end) {
		dayEnd := day.Add(24 * time.Hour)
		for _, r := range rules {
			if int(day.Weekday()) != r.Weekday {
				continue
			}
			if r.ValidFrom != nil && day.Before(*r.ValidFrom) {
				continue
			}
			if r.ValidUntil != nil && day.After(*r.ValidUntil) {
				continue
			}
			ws := parseHHMM(day, r.StartTime, loc)
			we := parseHHMM(day, r.EndTime, loc)
			if we.IsZero() || ws.IsZero() || !we.After(ws) {
				continue
			}
			// Clip to [from, to) and day boundaries.
			ws = maxTime(ws, from)
			ws = maxTime(ws, day)
			we = minTime(we, to)
			we = minTime(we, dayEnd)
			if we.After(ws) {
				out = append(out, interval{ws, we})
			}
		}
		day = dayEnd
	}
	// Merge overlapping windows from the same day.
	sortIntervals(out)
	return mergeIntervals(out)
}

func subtract(w interval, occupied []interval) []interval {
	result := []interval{w}
	for _, o := range occupied {
		if o.start.After(w.end) || !o.end.After(w.start) {
			continue // no overlap with original window, but result may vary
		}
		var next []interval
		for _, r := range result {
			next = append(next, subtractOne(r, o)...)
		}
		result = next
	}
	return result
}

func subtractOne(r, o interval) []interval {
	// No overlap
	if !o.end.After(r.start) || o.start.After(r.end) || o.start.Equal(r.end) {
		return []interval{r}
	}
	var out []interval
	if r.start.Before(o.start) {
		out = append(out, interval{r.start, o.start})
	}
	if o.end.Before(r.end) {
		out = append(out, interval{o.end, r.end})
	}
	return out
}

func sortIntervals(in []interval) {
	sort.Slice(in, func(i, j int) bool {
		return in[i].start.Before(in[j].start)
	})
}

func mergeIntervals(in []interval) []interval {
	if len(in) == 0 {
		return nil
	}
	out := []interval{in[0]}
	for _, iv := range in[1:] {
		last := &out[len(out)-1]
		if !iv.start.After(last.end) {
			if iv.end.After(last.end) {
				last.end = iv.end
			}
		} else {
			out = append(out, iv)
		}
	}
	return out
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// parseHHMM returns a time.Time on the same date as day but with time HH:MM.
func parseHHMM(day time.Time, hhmm string, loc *time.Location) time.Time {
	var h, m int
	fmt.Sscanf(hhmm, "%02d:%02d", &h, &m)
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, loc)
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
