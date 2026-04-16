package parser

import (
	"encoding/json"
	"os"
	"time"
)

// AuditEntry is the parsed representation of a ModSecurity JSON concurrent audit log file.
type AuditEntry struct {
	Transaction Transaction `json:"transaction"`
}

type Transaction struct {
	UniqueID   string  `json:"unique_id"`
	TimeStamp  string  `json:"time_stamp"`
	ClientIP   string  `json:"client_ip"`
	ClientPort int     `json:"client_port"`
	HostIP     string  `json:"host_ip"`
	HostPort   int     `json:"host_port"`

	Request  Request  `json:"request"`
	Response Response `json:"response"`
	Messages []Message `json:"messages"`
}

type Request struct {
	Method      string            `json:"method"`
	URI         string            `json:"uri"`
	HTTPVersion float64           `json:"http_version"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
}

type Response struct {
	HTTPCode int               `json:"http_code"`
	Headers  map[string]string `json:"headers"`
}

type Message struct {
	Message string  `json:"message"`
	Details Details `json:"details"`
}

type Details struct {
	RuleID   string   `json:"ruleId"`
	Severity string   `json:"severity"`
	Tags     []string `json:"tags"`
	Msg      string   `json:"match"` // ModSecurity uses "match" not "msg"
	Data     string   `json:"data"`
}

// ParseFile reads a single ModSecurity JSON concurrent audit log file.
func ParseFile(path string) (*AuditEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entry AuditEntry
	if err := json.NewDecoder(f).Decode(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// Timestamp parses the ModSecurity time_stamp field into a time.Time.
// ModSecurity format: "Thu Apr 16 17:33:12 2026"
func (t *Transaction) Timestamp() (time.Time, error) {
	return time.Parse("Mon Jan  2 15:04:05 2006", t.TimeStamp)
}
