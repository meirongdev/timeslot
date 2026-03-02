package models

import (
	"database/sql"
	"time"
)

// ---- Calendar ---------------------------------------------------------------

type Calendar struct {
	ID              int64
	Name            string
	Type            string // "ical_url" | "caldav"
	URL             string
	Credentials     string // encrypted blob
	SyncIntervalMin int
	LastSyncedAt    *time.Time
	Enabled         bool
}

type CalendarStore struct{ DB *sql.DB }

func (s *CalendarStore) List() ([]Calendar, error) {
	rows, err := s.DB.Query(`
		SELECT id, name, type, url, credentials,
		       sync_interval_min, last_synced_at, enabled
		FROM calendars ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Calendar
	for rows.Next() {
		var c Calendar
		var lastSync sql.NullTime
		var enabled int
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.Credentials,
			&c.SyncIntervalMin, &lastSync, &enabled); err != nil {
			return nil, err
		}
		if lastSync.Valid {
			c.LastSyncedAt = &lastSync.Time
		}
		c.Enabled = enabled == 1
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *CalendarStore) Get(id int64) (*Calendar, error) {
	var c Calendar
	var lastSync sql.NullTime
	var enabled int
	err := s.DB.QueryRow(`
		SELECT id, name, type, url, credentials,
		       sync_interval_min, last_synced_at, enabled
		FROM calendars WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Type, &c.URL, &c.Credentials,
			&c.SyncIntervalMin, &lastSync, &enabled)
	if err != nil {
		return nil, err
	}
	if lastSync.Valid {
		c.LastSyncedAt = &lastSync.Time
	}
	c.Enabled = enabled == 1
	return &c, nil
}

func (s *CalendarStore) Create(c *Calendar) error {
	res, err := s.DB.Exec(`
		INSERT INTO calendars (name, type, url, credentials, sync_interval_min, enabled)
		VALUES (?, ?, ?, ?, ?, ?)`,
		c.Name, c.Type, c.URL, c.Credentials, c.SyncIntervalMin, boolInt(c.Enabled))
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (s *CalendarStore) Update(c *Calendar) error {
	_, err := s.DB.Exec(`
		UPDATE calendars
		SET name=?, type=?, url=?, credentials=?, sync_interval_min=?, enabled=?
		WHERE id=?`,
		c.Name, c.Type, c.URL, c.Credentials, c.SyncIntervalMin, boolInt(c.Enabled), c.ID)
	return err
}

func (s *CalendarStore) Delete(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM calendars WHERE id = ?`, id)
	return err
}

func (s *CalendarStore) TouchSynced(id int64) error {
	_, err := s.DB.Exec(`UPDATE calendars SET last_synced_at=? WHERE id=?`, time.Now().UTC(), id)
	return err
}

// ---- BusyBlock --------------------------------------------------------------

type BusyBlock struct {
	ID         int64
	CalendarID int64
	StartAt    time.Time
	EndAt      time.Time
	Title      string
}

type BusyBlockStore struct{ DB *sql.DB }

// ListRange returns busy blocks overlapping [from, to).
func (s *BusyBlockStore) ListRange(from, to time.Time) ([]BusyBlock, error) {
	rows, err := s.DB.Query(`
		SELECT id, calendar_id, start_at, end_at, title
		FROM busy_blocks
		WHERE start_at < ? AND end_at > ?
		ORDER BY start_at`, to, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BusyBlock
	for rows.Next() {
		var b BusyBlock
		if err := rows.Scan(&b.ID, &b.CalendarID, &b.StartAt, &b.EndAt, &b.Title); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ReplaceForCalendar deletes all blocks for a calendar and inserts fresh ones.
func (s *BusyBlockStore) ReplaceForCalendar(calID int64, blocks []BusyBlock) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM busy_blocks WHERE calendar_id=?`, calID); err != nil {
		tx.Rollback()
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO busy_blocks (calendar_id,start_at,end_at,title) VALUES (?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, b := range blocks {
		if _, err = stmt.Exec(calID, b.StartAt.UTC(), b.EndAt.UTC(), b.Title); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ---- AvailabilityRule -------------------------------------------------------

type AvailabilityRule struct {
	ID         int64
	Weekday    int    // 0=Sun … 6=Sat
	StartTime  string // "HH:MM"
	EndTime    string // "HH:MM"
	ValidFrom  *time.Time
	ValidUntil *time.Time
}

type AvailabilityStore struct{ DB *sql.DB }

func (s *AvailabilityStore) List() ([]AvailabilityRule, error) {
	rows, err := s.DB.Query(`SELECT id,weekday,start_time,end_time,valid_from,valid_until FROM availability_rules ORDER BY weekday,start_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AvailabilityRule
	for rows.Next() {
		var r AvailabilityRule
		var vf, vu sql.NullTime
		if err := rows.Scan(&r.ID, &r.Weekday, &r.StartTime, &r.EndTime, &vf, &vu); err != nil {
			return nil, err
		}
		if vf.Valid {
			r.ValidFrom = &vf.Time
		}
		if vu.Valid {
			r.ValidUntil = &vu.Time
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *AvailabilityStore) Create(r *AvailabilityRule) error {
	res, err := s.DB.Exec(`INSERT INTO availability_rules (weekday,start_time,end_time,valid_from,valid_until) VALUES (?,?,?,?,?)`,
		r.Weekday, r.StartTime, r.EndTime, nullTime(r.ValidFrom), nullTime(r.ValidUntil))
	if err != nil {
		return err
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

func (s *AvailabilityStore) Update(r *AvailabilityRule) error {
	_, err := s.DB.Exec(`UPDATE availability_rules SET weekday=?,start_time=?,end_time=?,valid_from=?,valid_until=? WHERE id=?`,
		r.Weekday, r.StartTime, r.EndTime, nullTime(r.ValidFrom), nullTime(r.ValidUntil), r.ID)
	return err
}

func (s *AvailabilityStore) Delete(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM availability_rules WHERE id=?`, id)
	return err
}

// ---- helpers ----------------------------------------------------------------

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
