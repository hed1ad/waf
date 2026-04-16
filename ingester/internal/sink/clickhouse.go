package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/hediad/waf/ingester/internal/enrich"
	"github.com/hediad/waf/ingester/internal/parser"
)

type Event struct {
	Timestamp       time.Time
	TransactionID   string
	ClientIP        string
	ClientPort      uint16
	Method          string
	URI             string
	Protocol        string
	Host            string
	Status          uint16
	Action          string
	AnomalyScore    uint16
	RuleIDs         []uint32
	RuleMsgs        []string
	RuleTags        []string
	RuleSeverities  []string
	CountryCode     string
	CountryName     string
	City            string
	ASN             uint32
	AsnOrg          string
	RequestHeaders  string
	RequestBody     string
	ResponseHeaders string
}

type ClickHouseSink struct {
	conn driver.Conn
}

func NewClickHouseSink(ctx context.Context, dsn string) (*ClickHouseSink, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}
	return &ClickHouseSink{conn: conn}, nil
}

func (s *ClickHouseSink) Close() error {
	return s.conn.Close()
}

// BuildEvent converts a parsed audit entry + GeoIP info into a sink.Event.
func BuildEvent(entry *parser.AuditEntry, geo enrich.GeoInfo) Event {
	tx := entry.Transaction
	ts, _ := tx.Timestamp()
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	ev := Event{
		Timestamp:     ts,
		TransactionID: tx.UniqueID,
		ClientIP:      tx.ClientIP,
		ClientPort:    uint16(tx.ClientPort),
		Method:        tx.Request.Method,
		URI:           tx.Request.URI,
		Protocol:      fmt.Sprintf("HTTP/%.1f", tx.Request.HTTPVersion),
		Status:        uint16(tx.Response.HTTPCode),
		CountryCode:   geo.CountryCode,
		CountryName:   geo.CountryName,
		City:          geo.City,
		ASN:           geo.ASN,
		AsnOrg:        geo.AsnOrg,
		RequestBody:   tx.Request.Body,
	}

	if h, ok := tx.Request.Headers["host"]; ok {
		ev.Host = h
	}
	if h, ok := tx.Request.Headers["Host"]; ok {
		ev.Host = h
	}

	if b, err := json.Marshal(tx.Request.Headers); err == nil {
		ev.RequestHeaders = string(b)
	}
	if b, err := json.Marshal(tx.Response.Headers); err == nil {
		ev.ResponseHeaders = string(b)
	}

	ev.Action = "pass"

	for _, msg := range tx.Messages {
		id64, _ := strconv.ParseUint(msg.Details.RuleID, 10, 32)
		ev.RuleIDs = append(ev.RuleIDs, uint32(id64))
		ev.RuleMsgs = append(ev.RuleMsgs, msg.Details.Data)
		ev.RuleTags = append(ev.RuleTags, strings.Join(msg.Details.Tags, ","))
		ev.RuleSeverities = append(ev.RuleSeverities, msg.Details.Severity)
	}
	if ev.Status == 403 || ev.Status == 406 {
		ev.Action = "block"
	}

	return ev
}

// Insert writes a batch of events to ClickHouse in a single batch.
func (s *ClickHouseSink) Insert(ctx context.Context, events []Event) error {
	batch, err := s.conn.PrepareBatch(ctx, `INSERT INTO waf.events (
		timestamp, transaction_id, client_ip, client_port,
		method, uri, protocol, host, status,
		action, anomaly_score,
		rule_ids, rule_msgs, rule_tags, rule_severities,
		country_code, country_name, city, asn, asn_org,
		request_headers, request_body, response_headers
	)`)
	if err != nil {
		return err
	}

	for _, ev := range events {
		if err := batch.Append(
			ev.Timestamp, ev.TransactionID, ev.ClientIP, ev.ClientPort,
			ev.Method, ev.URI, ev.Protocol, ev.Host, ev.Status,
			ev.Action, ev.AnomalyScore,
			ev.RuleIDs, ev.RuleMsgs, ev.RuleTags, ev.RuleSeverities,
			ev.CountryCode, ev.CountryName, ev.City, ev.ASN, ev.AsnOrg,
			ev.RequestHeaders, ev.RequestBody, ev.ResponseHeaders,
		); err != nil {
			return err
		}
	}

	return batch.Send()
}
