// Package main adalah entrypoint Gateway Cloud — SaaS binary yang menginject
// extension points ke Gateway Core tanpa memodifikasi source code core.
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

	"go-gateway/handler"
	"go-gateway/hub"
	redisSub "go-gateway/redis"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := cloud.Load()

	// Logger
	level, err := zerolog.ParseLevel(cfg.Core.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	if cfg.Core.JWTSecret == "" {
		logger.Fatal().Msg("JWT_SECRET is required")
	}

	// Redis
	opt, err := goredis.ParseURL(cfg.Core.RedisURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("invalid REDIS_URL")
	}
	redisClient := goredis.NewClient(opt)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error().Err(err).Msg("redis ping failed")
	}

	// PostgreSQL — SaaS DB
	db := cloud.MustDB(cfg.DatabaseURL)
	defer db.Close()

	// Run migrations
	cloud.RunMigrations(db, logger)

	// Core hub
	h := hub.New(logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Redis subscriber
	go (redisSub.Subscriber{Client: redisClient, Hub: h, Log: logger}).Run(ctx)

	// Usage tracker — background flush goroutine
	usageTracker := cloud.NewUsageTracker(db)

	// Build SaaS extensions
	planLimiter := cloud.PlanRateLimiter{Redis: redisClient, DB: db}
	tenantAuth := cloud.TenantAuthenticator{DB: db}

	// HTTP routes
	mux := http.NewServeMux()

	// Core routes — dengan extension injection
	mux.Handle("/ws", handler.WSHandler{
		Config: cfg.Core, Hub: h, Log: logger,
		Auth: tenantAuth, RateLimiter: planLimiter, EventHook: usageTracker,
	})
	mux.Handle("/health", handler.HealthHandler{Hub: h, Redis: redisClient})
	mux.Handle("/api/socket/auth", handler.AuthHandler{
		Config: cfg.Core, Hub: h, Log: logger,
		Auth: tenantAuth, RateLimiter: planLimiter, EventHook: usageTracker,
	})
	mux.HandleFunc("/sdk/gateway.js", sdkHandler)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "gateway_connections %d\n", h.Connections())
	})

	// SaaS routes
	cloudAPI := cloud.NewAPI(db, cfg)
	mux.HandleFunc("/api/cloud/register", cloudAPI.Register)
	mux.HandleFunc("/api/cloud/usage", cloudAPI.Usage)
	mux.HandleFunc("/api/cloud/tenant", cloudAPI.GetTenant)
	mux.HandleFunc("/api/cloud/stripe/webhook", cloudAPI.StripeWebhook)

	// Server
	server := &http.Server{
		Addr:              ":" + cfg.Core.Port,
		Handler:           corsMiddleware(cfg.Core.AllowedOrigins, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info().Str("addr", server.Addr).Msg("gateway cloud server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("http server failed")
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info().Msg("gateway cloud shutting down")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("http shutdown failed")
	}
	if err := redisClient.Close(); err != nil {
		logger.Error().Err(err).Msg("redis close failed")
	}
}

func corsMiddleware(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, allowed, r.Header.Get("Origin"))
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

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

func sdkHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	_, _ = w.Write([]byte(`
class GatewayClient{constructor(o){this.o=o;this.ws=null;this.socketId=null;this.handlers={};this.channels=new Map();this.attempt=0;if(o.autoReconnect!==false)this.autoReconnect=true}on(e,h){(this.handlers[e]||(this.handlers[e]=new Set())).add(h);return this}off(e,h){if(this.handlers[e])this.handlers[e].delete(h);return this}bind(e,h){return this.on(e,h)}unbind(e,h){return this.off(e,h)}emit(e,d){(this.handlers[e]||[]).forEach(h=>{try{h(d)}catch(_){}})}connect(token){const host=this.o.host.replace(/^http/,'ws');this.ws=new WebSocket(host+'/ws?token='+encodeURIComponent(token));this.ws.onopen=()=>{this.attempt=0};this.ws.onmessage=e=>this.route(JSON.parse(e.data));this.ws.onclose=e=>{this.emit('disconnected',{reason:e.reason});if(this.autoReconnect)this.reconnect(token)};this.ws.onerror=()=>this.emit('error',{code:'WS_ERROR',message:'WebSocket error'})}disconnect(){this.autoReconnect=false;if(this.ws)this.ws.close()}subscribe(name,opt={}){const ch=new GatewayChannel(this,name);this.channels.set(name,ch);const send=a=>this.ws&&this.ws.readyState===1&&this.ws.send(JSON.stringify(a));if(name.startsWith('private-')||name.startsWith('presence-')){opt.auth({socket_id:this.socketId,channel_name:name}).then(r=>r.json?r.json():r).then(r=>send({type:'subscribe',channel:name,auth:r.data.auth,channel_data:r.data.channel_data}))}else send({type:'subscribe',channel:name});return ch}unsubscribe(n){if(this.ws)this.ws.send(JSON.stringify({type:'unsubscribe',channel:n}));this.channels.delete(n)}route(m){if(m.type==='system'){if(m.event==='connected'){this.socketId=m.data.socketId;this.emit('connected',m.data);this.channels.forEach((_,n)=>this.subscribe(n))}this.emit(m.event,m.data);const ch=this.channels.get(m.channel);if(ch)ch.emit(m.event,m.data);return}const ch=this.channels.get(m.channel);if(ch)ch.emit(m.event,m.data);this.emit(m.event,m.data)}reconnect(token){const d=Math.min(30000,1000*Math.pow(2,this.attempt++));this.emit('reconnecting',{attempt:this.attempt,delayMs:d});setTimeout(()=>this.connect(token),d)}}
class GatewayChannel{constructor(c,n){this.client=c;this.name=n;this.handlers={}}on(e,h){(this.handlers[e]||(this.handlers[e]=new Set())).add(h);return this}off(e,h){if(this.handlers[e])this.handlers[e].delete(h);return this}emit(e,d){(this.handlers[e]||[]).forEach(h=>h(d));(this.handlers['*']||[]).forEach(h=>h(e,d))}unsubscribe(){this.client.unsubscribe(this.name)}}
window.GatewayClient=GatewayClient;
`))
}
