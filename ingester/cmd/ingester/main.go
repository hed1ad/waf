package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hediad/waf/ingester/internal/enrich"
	"github.com/hediad/waf/ingester/internal/parser"
	"github.com/hediad/waf/ingester/internal/sink"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	auditDir   := envOr("AUDIT_LOG_DIR", "/var/log/modsecurity/audit")
	accessLog  := envOr("NGINX_ACCESS_LOG", "/var/log/nginx/access.log")
	chDSN      := envOr("CLICKHOUSE_DSN", "clickhouse://waf:wafpass@localhost:9000/waf")
	redisAddr  := envOr("REDIS_ADDR", "localhost:6379")
	geoipCity  := envOr("GEOIP_CITY_DB", "/data/GeoLite2-City.mmdb")
	geoipASN   := envOr("GEOIP_ASN_DB", "/data/GeoLite2-ASN.mmdb")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	chSink, err := sink.NewClickHouseSink(ctx, chDSN)
	if err != nil {
		log.Error("clickhouse connect failed", "err", err)
		os.Exit(1)
	}
	defer chSink.Close()

	redisSink := sink.NewRedisSink(redisAddr)
	defer redisSink.Close()

	var geo interface {
		Lookup(string) enrich.GeoInfo
		Close()
	}
	enricher, err := enrich.NewGeoEnricher(geoipCity, geoipASN)
	if err != nil {
		log.Warn("geoip unavailable, enrichment disabled", "err", err)
		geo = enrich.NoopEnricher{}
	} else {
		geo = enricher
		defer enricher.Close()
	}

	// ── Batch flusher ────────────────────────────────────────────────────────
	batchCh := make(chan sink.Event, 512)
	go func() {
		var batch []sink.Event
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case ev, ok := <-batchCh:
				if !ok {
					return
				}
				batch = append(batch, ev)
			case <-ticker.C:
				if len(batch) == 0 {
					continue
				}
				if err := chSink.Insert(ctx, batch); err != nil {
					log.Error("clickhouse insert failed", "count", len(batch), "err", err)
				} else {
					log.Info("flushed events", "count", len(batch))
				}
				batch = batch[:0]
			case <-ctx.Done():
				if len(batch) > 0 {
					chSink.Insert(ctx, batch)
				}
				return
			}
		}
	}()

	// ── Nginx access log tailer (pass events) ─────────────────────────────────
	go tailAccessLog(ctx, accessLog, geo, batchCh, redisSink, log)

	// ── ModSecurity audit log watcher (block events) ─────────────────────────
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("fsnotify init failed", "err", err)
		os.Exit(1)
	}
	defer watcher.Close()

	if err := addRecursive(watcher, auditDir); err != nil {
		log.Error("cannot watch audit dir", "dir", auditDir, "err", err)
		os.Exit(1)
	}

	log.Info("ingester started", "audit_dir", auditDir, "access_log", accessLog)

	for {
		select {
		case <-ctx.Done():
			log.Info("ingester stopped")
			return

		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					watcher.Add(ev.Name)
					continue
				}
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			name := filepath.Base(ev.Name)
			if name == "audit.log" || filepath.Ext(name) == ".log" {
				continue
			}

			entry, err := parser.ParseFile(ev.Name)
			if err != nil {
				log.Warn("parse failed", "file", ev.Name, "err", err)
				continue
			}

			geoInfo := geo.Lookup(entry.Transaction.ClientIP)
			event := sink.BuildEvent(entry, geoInfo)
			batchCh <- event

			if pubErr := redisSink.Publish(ctx, event); pubErr != nil {
				log.Warn("redis publish failed", "err", pubErr)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error("watcher error", "err", err)
		}
	}
}

// tailAccessLog tails the nginx access log and sends pass events to batchCh.
func tailAccessLog(ctx context.Context, path string, geo interface{ Lookup(string) enrich.GeoInfo }, batchCh chan<- sink.Event, redisSink *sink.RedisSink, log *slog.Logger) {
	// Wait for the file to appear
	for {
		if _, err := os.Stat(path); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}

	f, err := os.Open(path)
	if err != nil {
		log.Warn("cannot open access log", "path", path, "err", err)
		return
	}
	defer f.Close()

	// Seek to end — only process new entries
	f.Seek(0, io.SeekEnd)

	entryCh := make(chan *parser.AccessEntry, 256)
	go parser.ScanAccessLog(f, entryCh)

	for {
		select {
		case <-ctx.Done():
			return
		case ae, ok := <-entryCh:
			if !ok {
				return
			}
			geoInfo := geo.Lookup(ae.ClientIP)
			ev := sink.Event{
				Timestamp:     ae.Timestamp,
				TransactionID: fmt.Sprintf("access-%d-%s", ae.Timestamp.UnixMilli(), ae.ClientIP),
				ClientIP:      ae.ClientIP,
				Method:        ae.Method,
				URI:           ae.URI,
				Host:          ae.Host,
				Protocol:      ae.Protocol,
				Status:        uint16(ae.Status),
				Action:        "pass",
				CountryCode:   geoInfo.CountryCode,
				CountryName:   geoInfo.CountryName,
				City:          geoInfo.City,
				ASN:           geoInfo.ASN,
				AsnOrg:        geoInfo.AsnOrg,
			}
			batchCh <- ev
			if pubErr := redisSink.Publish(ctx, ev); pubErr != nil {
				log.Warn("redis publish failed", "err", pubErr)
			}
		}
	}
}

func addRecursive(w *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return w.Add(path)
		}
		return nil
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
