package cloud

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// UsageTracker mengimplementasikan extensions.EventHook.
// Setiap event lifecycle dicatat ke usage_events untuk billing per-tenant.
type UsageTracker struct {
	DB     *sql.DB
	Buffer chan UsageEvent
}

// UsageEvent adalah satu event yang akan di-persist async.
type UsageEvent struct {
	TenantID     string
	EventType    string // connect, disconnect, subscribe, unsubscribe, publish
	Channel      string
	PayloadBytes int64
	Timestamp    time.Time
}

// NewUsageTracker membuat UsageTracker dengan buffered channel.
// Goroutine background flush event ke database setiap batch atau interval.
func NewUsageTracker(db *sql.DB) *UsageTracker {
	t := &UsageTracker{
		DB:     db,
		Buffer: make(chan UsageEvent, 1000),
	}
	go t.flush()
	return t
}

func (t *UsageTracker) OnConnect(tenantID, socketID string) {
	t.track(tenantID, "connect", "", 0)
}

func (t *UsageTracker) OnDisconnect(tenantID, socketID string) {
	t.track(tenantID, "disconnect", "", 0)
}

func (t *UsageTracker) OnSubscribe(tenantID, channel, socketID string) {
	t.track(tenantID, "subscribe", channel, 0)
}

func (t *UsageTracker) OnUnsubscribe(tenantID, channel, socketID string) {
	t.track(tenantID, "unsubscribe", channel, 0)
}

func (t *UsageTracker) OnPublish(tenantID, channel, event string, payloadSize int64) {
	t.track(tenantID, "publish", channel, payloadSize)
}

func (t *UsageTracker) track(tenantID, eventType, channel string, payloadBytes int64) {
	if tenantID == "" {
		return
	}
	select {
	case t.Buffer <- UsageEvent{
		TenantID:     tenantID,
		EventType:    eventType,
		Channel:      channel,
		PayloadBytes: payloadBytes,
		Timestamp:    time.Now(),
	}:
	default:
		// Buffer penuh — drop event agar tidak memblokir path critical.
	}
}

// flush menulis event dari buffer ke database secara batch.
func (t *UsageTracker) flush() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var batch []UsageEvent
	for {
		select {
		case ev := <-t.Buffer:
			batch = append(batch, ev)
			if len(batch) >= 100 {
				t.insertBatch(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				t.insertBatch(batch)
				batch = nil
			}
		}
	}
}

func (t *UsageTracker) insertBatch(events []UsageEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("usage: begin tx failed: %v", err)
		return
	}
	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO usage_events (tenant_id, event_type, channel, payload_bytes, created_at) VALUES ($1, $2, $3, $4, $5)",
	)
	if err != nil {
		tx.Rollback()
		log.Printf("usage: prepare stmt failed: %v", err)
		return
	}
	defer stmt.Close()
	for _, ev := range events {
		_, err := stmt.ExecContext(ctx, ev.TenantID, ev.EventType, ev.Channel, ev.PayloadBytes, ev.Timestamp)
		if err != nil {
			log.Printf("usage: insert failed: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("usage: commit failed: %v", err)
	}
}
