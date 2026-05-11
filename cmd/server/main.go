// Package main adalah entrypoint Gateway Cloud — SaaS binary yang menginject
// extension points ke Gateway Core tanpa memodifikasi source code core.
//
// Arsitektur:
//
//	Gateway Cloud (binary ini)
//	├── Core Gateway (go-gateway)  ← open source, diimpor sebagai Go module
//	│   ├── handler.WSHandler      ← diinject Auth, RateLimiter, EventHook
//	│   ├── handler.AuthHandler    ← diinject Auth, RateLimiter, EventHook
//	│   ├── handler.HealthHandler  ← tetap sama, tidak butuh extension
//	│   └── hub, redis, auth       ← core logic, tidak diubah
//	├── SaaS Extensions (gateway_cloud/cloud)
//	│   ├── TenantAuthenticator    ← validasi X-Tenant-Key + X-User-ID
//	│   ├── PlanRateLimiter        ← Redis token bucket per plan tier
//	│   └── UsageTracker           ← event lifecycle → PostgreSQL
//	└── SaaS API (gateway_cloud/cloud)
//	    ├── POST /api/cloud/register       ← daftar tenant baru
//	    ├── GET  /api/cloud/usage          ← statistik usage per tenant
//	    ├── GET  /api/cloud/tenant         ← detail tenant via API key
//	    └── POST /api/cloud/stripe/webhook ← Stripe events → update plan
//
// Build:
//
//	go build -o gateway-cloud ./cmd/server
//
// Run:
//
//	DATABASE_URL=postgres://... REDIS_URL=redis://... JWT_SECRET=... ./gateway-cloud
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gateway_cloud/cloud"

	// Module core Gateway Realtime — diimpor utuh tanpa modifikasi.
	"go-gateway/handler"
	"go-gateway/hub"
	redisSub "go-gateway/redis"

	// Dependencies sama dengan core — Redis dan logger.
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// main adalah entrypoint binary SaaS. Alurnya:
//
//	1. Load config (core + SaaS env)
//	2. Connect Redis + PostgreSQL
//	3. Run auto-migration
//	4. Init core Hub + Redis subscriber
//	5. Build SaaS extensions (auth, rate limit, usage tracker)
//	6. Mount core routes + SaaS routes di HTTP mux
//	7. Start HTTP server + graceful shutdown
func main() {
	// Load config — menggabungkan env core Gateway + env SaaS.
	cfg := cloud.Load()

	// Logger — sama dengan core: zerolog + console writer.
	level, err := zerolog.ParseLevel(cfg.Core.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	// JWT secret wajib — dipakai untuk sign token WebSocket dan auth endpoint.
	if cfg.Core.JWTSecret == "" {
		logger.Fatal().Msg("JWT_SECRET is required — set a random 64-char string")
	}

	// --- Infrastructure: Redis ---
	// Redis digunakan untuk: (1) pub/sub core Gateway, (2) rate limiting SaaS.
	opt, err := goredis.ParseURL(cfg.Core.RedisURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("invalid REDIS_URL")
	}
	redisClient := goredis.NewClient(opt)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error().Err(err).Msg("redis ping failed — continuing, but pub/sub may not work")
	}

	// --- Infrastructure: PostgreSQL ---
	// PostgreSQL menyimpan tenants, usage events, dan nantinya billing data.
	db := cloud.MustDB(cfg.DatabaseURL)
	defer db.Close()

	// Auto-migration: buat tabel jika belum ada (idempoten).
	cloud.RunMigrations(db, logger)

	// --- Core: Gateway Hub + Redis Subscriber ---
	// Hub adalah registry global koneksi WebSocket dari core.
	h := hub.New(logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Redis subscriber: mendengarkan pesan pub/sub dan mem-fanout ke Hub.
	go (redisSub.Subscriber{Client: redisClient, Hub: h, Log: logger}).Run(ctx)

	// --- SaaS: Usage Tracker ---
	// Background goroutine yang flush usage events ke PostgreSQL.
	usageTracker := cloud.NewUsageTracker(db)

	// --- SaaS: Rate Limiter & Tenant Authenticator ---
	// PlanRateLimiter menentukan apakah request diterima berdasarkan plan tenant.
	planLimiter := cloud.PlanRateLimiter{Redis: redisClient, DB: db}
	// TenantAuthenticator memvalidasi header X-Tenant-Key dan X-User-ID.
	tenantAuth := cloud.TenantAuthenticator{DB: db}

	// --- HTTP Routes ---
	mux := http.NewServeMux()

	// Core routes — sama dengan open source, tapi diinject SaaS extensions.
	// Handler akan menggunakan extensions.Authenticator, .RateLimiter, .EventHook
	// yang sudah diinject; jika kosong, fallback ke default no-op.
	mux.Handle("/ws", handler.WSHandler{
		Config:      cfg.Core,
		Hub:         h,
		Log:         logger,
		Auth:        tenantAuth,    // SaaS: validasi tenant + user.
		RateLimiter: planLimiter,   // SaaS: batasi koneksi per plan.
		EventHook:   usageTracker,  // SaaS: catat connect/disconnect.
	})
	mux.Handle("/health", handler.HealthHandler{Hub: h, Redis: redisClient})
	mux.Handle("/api/socket/auth", handler.AuthHandler{
		Config:      cfg.Core,
		Hub:         h,
		Log:         logger,
		Auth:        tenantAuth,    // SaaS: validasi sebelum sign channel.
		RateLimiter: planLimiter,   // SaaS: batasi auth request per tenant.
		EventHook:   usageTracker,  // SaaS: catat subscribe event.
	})
	// Browser SDK — disajikan sebagai static JS dari memory.
	mux.HandleFunc("/sdk/gateway.js", sdkHandler)
	// Prometheus metrics — kompatibel dengan core format.
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "gateway_connections %d\n", h.Connections())
	})

	// --- SaaS Routes ---
	// Endpoint SaaS: tenant management, usage analytics, Stripe webhook.
	cloudAPI := cloud.NewAPI(db, cfg)
	mux.HandleFunc("/api/cloud/register", cloudAPI.Register)
	mux.HandleFunc("/api/cloud/usage", cloudAPI.Usage)
	mux.HandleFunc("/api/cloud/tenant", cloudAPI.GetTenant)
	mux.HandleFunc("/api/cloud/stripe/webhook", cloudAPI.StripeWebhook)

	// --- HTTP Server ---
	server := &http.Server{
		Addr:              ":" + cfg.Core.Port,
		Handler:           corsMiddleware(cfg.Core.AllowedOrigins, mux),
		ReadHeaderTimeout: 10 * time.Second, // Mitigasi slow-loris.
	}

	// Start HTTP server di goroutine terpisah.
	go func() {
		logger.Info().Str("addr", server.Addr).Msg("gateway cloud server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("http server failed")
		}
	}()

	// --- Graceful Shutdown ---
	// Block sampai SIGINT atau SIGTERM, lalu shutdown server dan tutup koneksi.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info().Msg("gateway cloud shutting down")
	cancel() // Hentikan goroutine Redis subscriber.

	// Timeout 5 detik untuk shutdown HTTP server.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("http shutdown failed")
	}
	if err := redisClient.Close(); err != nil {
		logger.Error().Err(err).Msg("redis close failed")
	}
}

// corsMiddleware memasang header CORS global sebelum request masuk ke handler.
// Preflight OPTIONS dijawab langsung tanpa melewati handler untuk efisiensi.
func corsMiddleware(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, allowed, r.Header.Get("Origin"))
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// setCORSHeaders memilih Access-Control-Allow-Origin berdasarkan allowlist.
// Wildcard (*) didukung untuk development, tapi production sebaiknya explicit origin.
// Header tambahan X-Tenant-Key dan X-User-ID untuk SaaS authentication.
func setCORSHeaders(w http.ResponseWriter, allowed []string, origin string) {
	for _, candidate := range allowed {
		if candidate == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			break
		}
		if candidate == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
	}
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-Key, X-User-ID")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

// sdkHandler menyajikan GatewayClient browser SDK dari memory.
// SDK di-embed sebagai string literal agar binary SaaS tetap self-contained.
// Kode JavaScript ini identik dengan SDK yang disajikan oleh Gateway Core.
func sdkHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	_, _ = w.Write([]byte(`
class GatewayClient{constructor(o){this.o=o;this.ws=null;this.socketId=null;this.handlers={};this.channels=new Map();this.attempt=0;if(o.autoReconnect!==false)this.autoReconnect=true}on(e,h){(this.handlers[e]||(this.handlers[e]=new Set())).add(h);return this}off(e,h){if(this.handlers[e])this.handlers[e].delete(h);return this}bind(e,h){return this.on(e,h)}unbind(e,h){return this.off(e,h)}emit(e,d){(this.handlers[e]||[]).forEach(h=>{try{h(d)}catch(_){}})}connect(token){const host=this.o.host.replace(/^http/,'ws');this.ws=new WebSocket(host+'/ws?token='+encodeURIComponent(token));this.ws.onopen=()=>{this.attempt=0};this.ws.onmessage=e=>this.route(JSON.parse(e.data));this.ws.onclose=e=>{this.emit('disconnected',{reason:e.reason});if(this.autoReconnect)this.reconnect(token)};this.ws.onerror=()=>this.emit('error',{code:'WS_ERROR',message:'WebSocket error'})}disconnect(){this.autoReconnect=false;if(this.ws)this.ws.close()}subscribe(name,opt={}){const ch=new GatewayChannel(this,name);this.channels.set(name,ch);const send=a=>this.ws&&this.ws.readyState===1&&this.ws.send(JSON.stringify(a));if(name.startsWith('private-')||name.startsWith('presence-')){opt.auth({socket_id:this.socketId,channel_name:name}).then(r=>r.json?r.json():r).then(r=>send({type:'subscribe',channel:name,auth:r.data.auth,channel_data:r.data.channel_data}))}else send({type:'subscribe',channel:name});return ch}unsubscribe(n){if(this.ws)this.ws.send(JSON.stringify({type:'unsubscribe',channel:n}));this.channels.delete(n)}route(m){if(m.type==='system'){if(m.event==='connected'){this.socketId=m.data.socketId;this.emit('connected',m.data);this.channels.forEach((_,n)=>this.subscribe(n))}this.emit(m.event,m.data);const ch=this.channels.get(m.channel);if(ch)ch.emit(m.event,m.data);return}const ch=this.channels.get(m.channel);if(ch)ch.emit(m.event,m.data);this.emit(m.event,m.data)}reconnect(token){const d=Math.min(30000,1000*Math.pow(2,this.attempt++));this.emit('reconnecting',{attempt:this.attempt,delayMs:d});setTimeout(()=>this.connect(token),d)}}
class GatewayChannel{constructor(c,n){this.client=c;this.name=n;this.handlers={}}on(e,h){(this.handlers[e]||(this.handlers[e]=new Set())).add(h);return this}off(e,h){if(this.handlers[e])this.handlers[e].delete(h);return this}emit(e,d){(this.handlers[e]||[]).forEach(h=>h(d));(this.handlers['*']||[]).forEach(h=>h(e,d))}unsubscribe(){this.client.unsubscribe(this.name)}}
window.GatewayClient=GatewayClient;
`))
}
