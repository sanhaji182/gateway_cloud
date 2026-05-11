module gateway_cloud

go 1.22

require (
	github.com/lib/pq v1.12.3
	github.com/redis/go-redis/v9 v9.7.0
	github.com/rs/zerolog v1.33.0
	go-gateway v0.1.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

replace go-gateway => ../gateway/backend_go
