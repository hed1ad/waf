package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type EventFilter struct {
	From      time.Time
	To        time.Time
	Action    string // '' = all, 'block', 'pass'
	ClientIP  string
	Country   string
	RuleID    uint32
	Limit     int
	Offset    int
}

type Event struct {
	Timestamp     time.Time `json:"timestamp"`
	TransactionID string    `json:"transaction_id"`
	ClientIP      string    `json:"client_ip"`
	Method        string    `json:"method"`
	URI           string    `json:"uri"`
	Host          string    `json:"host"`
	Status        uint16    `json:"status"`
	Action        string    `json:"action"`
	AnomalyScore  uint16    `json:"anomaly_score"`
	CountryCode   string    `json:"country_code"`
	CountryName   string    `json:"country_name"`
	RuleIDs       []uint32  `json:"rule_ids"`
	RuleMsgs      []string  `json:"rule_msgs"`
}

type StatsPoint struct {
	Minute  time.Time `json:"minute"`
	Total   uint64    `json:"total"`
	Blocked uint64    `json:"blocked"`
}

type TopIP struct {
	ClientIP    string `json:"client_ip"`
	CountryCode string `json:"country_code"`
	TotalHits   uint64 `json:"total_hits"`
	BlockedHits uint64 `json:"blocked_hits"`
	LastSeen    time.Time `json:"last_seen"`
}

type ClickHouseStore struct {
	conn driver.Conn
}

func NewClickHouseStore(ctx context.Context, dsn string) (*ClickHouseStore, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, err
	}
	return &ClickHouseStore{conn: conn}, nil
}

func (s *ClickHouseStore) Close() error { return s.conn.Close() }

func (s *ClickHouseStore) ListEvents(ctx context.Context, f EventFilter) ([]Event, error) {
	if f.Limit == 0 {
		f.Limit = 50
	}
	rows, err := s.conn.Query(ctx, `
		SELECT timestamp, transaction_id, client_ip, method, uri, host,
		       status, action, anomaly_score, country_code, country_name,
		       rule_ids, rule_msgs
		FROM waf.events
		WHERE timestamp BETWEEN ? AND ?
		  AND (? = '' OR action = ?)
		  AND (? = '' OR client_ip = ?)
		  AND (? = '' OR country_code = ?)
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`,
		f.From, f.To,
		f.Action, f.Action,
		f.ClientIP, f.ClientIP,
		f.Country, f.Country,
		f.Limit, f.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var ev Event
		if err := rows.Scan(
			&ev.Timestamp, &ev.TransactionID, &ev.ClientIP, &ev.Method,
			&ev.URI, &ev.Host, &ev.Status, &ev.Action, &ev.AnomalyScore,
			&ev.CountryCode, &ev.CountryName, &ev.RuleIDs, &ev.RuleMsgs,
		); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (s *ClickHouseStore) GetTimeseries(ctx context.Context, from, to time.Time) ([]StatsPoint, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT minute, sum(total) AS total, sum(blocked) AS blocked
		FROM waf.events_1m
		WHERE minute BETWEEN ? AND ?
		GROUP BY minute
		ORDER BY minute`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pts []StatsPoint
	for rows.Next() {
		var p StatsPoint
		if err := rows.Scan(&p.Minute, &p.Total, &p.Blocked); err != nil {
			return nil, err
		}
		pts = append(pts, p)
	}
	return pts, rows.Err()
}

func (s *ClickHouseStore) GetTopIPs(ctx context.Context, limit int) ([]TopIP, error) {
	if limit == 0 {
		limit = 10
	}
	rows, err := s.conn.Query(ctx, `
		SELECT client_ip, anyLast(country_code), sum(total_hits), sum(blocked_hits), max(last_seen)
		FROM waf.top_ips
		GROUP BY client_ip
		ORDER BY sum(total_hits) DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []TopIP
	for rows.Next() {
		var ip TopIP
		if err := rows.Scan(&ip.ClientIP, &ip.CountryCode, &ip.TotalHits, &ip.BlockedHits, &ip.LastSeen); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

func (s *ClickHouseStore) CountEvents(ctx context.Context, from, to time.Time) (total, blocked uint64, err error) {
	row := s.conn.QueryRow(ctx, `
		SELECT count(), countIf(action='block')
		FROM waf.events
		WHERE timestamp BETWEEN ? AND ?`, from, to)
	err = row.Scan(&total, &blocked)
	return
}
