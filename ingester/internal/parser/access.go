package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

// nginx log format from owasp/modsecurity-crs image:
// $host $remote_addr - [$time_local] "$request" $status $body_bytes_sent $http_referer "$http_user_agent" $unique_id ...
var accessLogRe = regexp.MustCompile(
	`^(\S+)\s+(\S+)\s+-\s+\[([^\]]+)\]\s+"(\w+)\s+(\S+)\s+HTTP/([\d.]+)"\s+(\d+)\s+\d+`,
)

type AccessEntry struct {
	Timestamp time.Time
	ClientIP  string
	Host      string
	Method    string
	URI       string
	Protocol  string
	Status    int
}

func ParseAccessLine(line string) (*AccessEntry, error) {
	m := accessLogRe.FindStringSubmatch(line)
	if m == nil {
		return nil, fmt.Errorf("no match")
	}
	// format: 02/Jan/2006:15:04:05 +0000
	ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[3])
	if err != nil {
		ts = time.Now()
	}
	status, _ := strconv.Atoi(m[7])
	return &AccessEntry{
		Timestamp: ts,
		Host:      m[1],
		ClientIP:  m[2],
		Method:    m[4],
		URI:       m[5],
		Protocol:  "HTTP/" + m[6],
		Status:    status,
	}, nil
}

// ScanAccessLog reads lines from r and sends parsed non-blocked entries to out.
// 403/406 are skipped — they are already captured by the ModSecurity audit log.
func ScanAccessLog(r io.Reader, out chan<- *AccessEntry) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		entry, err := ParseAccessLine(scanner.Text())
		if err != nil {
			continue
		}
		if entry.Status == 403 || entry.Status == 406 {
			continue // block events come from the audit log with full rule details
		}
		out <- entry
	}
}
