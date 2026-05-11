package cloud

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// UsageTracker mengimplementasikan extensions.EventHook dari core.
//
// Setiap event lifecycle (connect, disconnect, subscribe, unsubscribe, publish)
// dari Gateway Core dikirim ke buffer channel, lalu di-flush ke PostgreSQL
// secara batch setiap 5 detik atau setiap 100 event — mana yang tercapai dulu.
//
// Arsitektur async ini memastikan pencatatan usage TIDAK memblokir path critical
// WebSocket dan REST API. Jika buffer penuh (1000 events), event baru di-drop
// agar tidak terjadi backpressure ke core Gateway.
type UsageTracker struct {
	DB     *sql.DB       // Koneksi PostgreSQL untuk insert usage_events.
	Buffer chan UsageEvent // Buffered channel (capacity 1000) untuk pipeline async.
}

// UsageEvent adalah satu record yang akan di-persist ke usage_events.
// Fields mengikuti schema tabel usage_events untuk batch insert efisien.
type UsageEvent struct {
	TenantID     string    // UUID tenant — kosong untuk self-hosted.
	EventType    string    // connect, disconnect, subscribe, unsubscribe, publish.
	Channel      string    // Nama channel (kosong untuk event koneksi).
	PayloadBytes int64     // Ukuran payload dalam bytes (hanya untuk publish).
	Timestamp    time.Time // Waktu event terjadi — untuk query analytics per periode.
}

// NewUsageTracker membuat UsageTracker dan menjalankan goroutine flush background.
// Goroutine ini berjalan sepanjang hidup aplikasi dan berhenti saat proses mati.
func NewUsageTracker(db *sql.DB) *UsageTracker {
	t := &UsageTracker{
		DB:     db,
		Buffer: make(chan UsageEvent, 1000), // Buffer 1000 events ≈ beberapa detik traffic peak.
	}
	go t.flush() // Goroutine background untuk batch insert.
	return t
}

// OnConnect dipanggil setiap WebSocket berhasil upgrade.
// Hanya dicatat jika tenantID tidak kosong (SaaS user).
func (t *UsageTracker) OnConnect(tenantID, socketID string) {
	t.track(tenantID, "connect", "", 0)
}

// OnDisconnect dipanggil setiap WebSocket connection terputus.
func (t *UsageTracker) OnDisconnect(tenantID, socketID string) {
	t.track(tenantID, "disconnect", "", 0)
}

// OnSubscribe dipanggil setiap client subscribe ke channel.
func (t *UsageTracker) OnSubscribe(tenantID, channel, socketID string) {
	t.track(tenantID, "subscribe", channel, 0)
}

// OnUnsubscribe dipanggil setiap client unsubscribe dari channel.
func (t *UsageTracker) OnUnsubscribe(tenantID, channel, socketID string) {
	t.track(tenantID, "unsubscribe", channel, 0)
}

// OnPublish dipanggil setiap event dipublish ke channel.
// payloadSize dicatat untuk billing berbasis volume data.
func (t *UsageTracker) OnPublish(tenantID, channel, event string, payloadSize int64) {
	t.track(tenantID, "publish", channel, payloadSize)
}

// track mengirim event ke buffer channel secara non-blocking.
// Jika tenantID kosong (self-hosted), event diabaikan.
// Jika buffer penuh, event di-drop — prioritas: jangan blokir core Gateway.
func (t *UsageTracker) track(tenantID, eventType, channel string, payloadBytes int64) {
	if tenantID == "" {
		return // Self-hosted users tidak ditracking.
	}
	select {
	case t.Buffer <- UsageEvent{
		TenantID:     tenantID,
		EventType:    eventType,
		Channel:      channel,
		PayloadBytes: payloadBytes,
		Timestamp:    time.Now(),
	}:
		// Event berhasil masuk buffer — akan di-flush dalam batch.
	default:
		// Buffer penuh — drop event agar tidak memblokir goroutine core.
		// Production monitoring harus alert jika ini sering terjadi.
	}
}

// flush membaca dari buffer channel dan menulis ke PostgreSQL secara batch.
// Batch dikirim setiap 5 detik ATAU setiap 100 event terkumpul —
// mana yang tercapai dulu. Ini menyeimbangkan latency dengan throughput.
func (t *UsageTracker) flush() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var batch []UsageEvent
	for {
		select {
		case ev := <-t.Buffer:
			batch = append(batch, ev)
			// Batch berukuran 100 untuk efisiensi insert tanpa memblokir buffer terlalu lama.
			if len(batch) >= 100 {
				t.insertBatch(batch)
				batch = nil
			}
		case <-ticker.C:
			// Flush sisa event setiap 5 detik agar tidak ada yang tertinggal di buffer.
			if len(batch) > 0 {
				t.insertBatch(batch)
				batch = nil
			}
		}
	}
}

// insertBatch menulis slice UsageEvent ke database dalam satu transaksi.
// Menggunakan prepared statement untuk efisiensi dan transaksi untuk atomicity.
// Error dilog — tidak di-return karena ini goroutine background.
func (t *UsageTracker) insertBatch(events []UsageEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("usage: begin tx failed: %v", err)
		return
	}
	// Prepared statement untuk insert massal — compiled sekali per batch.
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
			log.Printf("usage: insert failed for tenant %s event %s: %v", ev.TenantID, ev.EventType, err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("usage: commit failed: %v", err)
	}
}
