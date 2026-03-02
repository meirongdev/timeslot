package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/meirongdev/timeslot/admin"
	"github.com/meirongdev/timeslot/api"
	"github.com/meirongdev/timeslot/config"
	"github.com/meirongdev/timeslot/db"
	"github.com/meirongdev/timeslot/engine"
	"github.com/meirongdev/timeslot/models"
	"github.com/meirongdev/timeslot/sync"
)

func main() {
	cfgPath := flag.String("config", "config.json", "path to JSON config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// ---- database -----------------------------------------------------------
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	// ---- stores -------------------------------------------------------------
	calStore := &models.CalendarStore{DB: database}
	busyStore := &models.BusyBlockStore{DB: database}
	availStore := &models.AvailabilityStore{DB: database}

	// ---- location -----------------------------------------------------------
	loc := time.UTC
	if cfg.Timezone != "" {
		if l, err := time.LoadLocation(cfg.Timezone); err == nil {
			loc = l
		} else {
			log.Printf("unknown timezone %q, using UTC: %v", cfg.Timezone, err)
		}
	}

	// ---- slot engine --------------------------------------------------------
	eng := &engine.Engine{
		Availability: availStore,
		Busy:         busyStore,
		Location:     loc,
		SlotDuration: time.Duration(cfg.SlotDurationMin) * time.Minute,
		BufferBefore: time.Duration(cfg.BufferBeforeMin) * time.Minute,
	}

	// ---- sync worker --------------------------------------------------------
	worker := &sync.Worker{
		Calendars:  calStore,
		BusyBlocks: busyStore,
	}

	// ---- cron ---------------------------------------------------------------
	c := cron.New()
	c.AddFunc("@every 15m", worker.SyncAll)
	c.Start()
	defer c.Stop()

	// Do an initial sync on startup.
	go worker.SyncAll()

	// ---- HTTP routing -------------------------------------------------------
	mux := http.NewServeMux()

	// Public API
	apiH := &api.Handler{
		Cfg:    cfg,
		Engine: eng,
	}
	apiH.RegisterRoutes(mux)

	// ---- Admin UI -----------------------------------------------------------
	adminH := &admin.Handler{
		Cfg:          cfg,
		Calendars:    calStore,
		Availability: availStore,
		BusyBlocks:   busyStore,
		SyncWorker:   worker,
		TemplateDir:  "templates",
	}
	adminH.RegisterRoutes(mux)

	// Static fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})

	log.Printf("timeslot listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
