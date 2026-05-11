package cloud

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	// lib/pq adalah pure-Go PostgreSQL driver (database/sql compatible).
	// Dipilih karena ringan, tidak butuh CGO, dan bekerja baik di Alpine Docker.
	_ "github.com/lib/pq"
)

// MustDB membuka koneksi PostgreSQL atau keluar dengan os.Exit(1) jika gagal.
// Fungsi ini hanya dipanggil saat startup; jika DATABASE_URL kosong atau
// PostgreSQL tidak bisa dijangkau, aplikasi tidak akan berjalan (fail-fast).
//
// Konfigurasi pool:
//   - MaxOpenConns: 25 (cukup untuk throughput menengah SaaS)
//   - MaxIdleConns: 5  (menjaga koneksi siap tanpa membebani PostgreSQL)
func MustDB(databaseURL string) *sql.DB {
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required — set PostgreSQL connection string")
		os.Exit(1)
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	if err := db.PingContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping database: %v\n", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	fmt.Println("PostgreSQL connected")
	return db
}
