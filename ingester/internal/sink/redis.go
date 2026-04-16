package sink

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const streamChannel = "waf:events:live"

type RedisSink struct {
	client *redis.Client
}

func NewRedisSink(addr string) *RedisSink {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	return &RedisSink{client: client}
}

func (r *RedisSink) Close() error {
	return r.client.Close()
}

// Publish sends a lightweight event summary to the live-stream channel.
func (r *RedisSink) Publish(ctx context.Context, ev Event) error {
	payload := map[string]any{
		"id":         ev.TransactionID,
		"ts":         ev.Timestamp.UnixMilli(),
		"ip":         ev.ClientIP,
		"method":     ev.Method,
		"uri":        ev.URI,
		"status":     ev.Status,
		"action":     ev.Action,
		"country":    ev.CountryCode,
		"rule_count": len(ev.RuleIDs),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, streamChannel, b).Err()
}
