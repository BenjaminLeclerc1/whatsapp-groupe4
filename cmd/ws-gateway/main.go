package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/whatsapp-groupe4/internal/logger"
	"github.com/whatsapp-groupe4/internal/middleware"
	"github.com/whatsapp-groupe4/internal/wsgateway"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func main() {
	logger.Init("ws-gateway")
	defer logger.Close()

	raiseFileLimit()

	port := getEnv("PORT", "8089")
	jwtSecret := requireEnv("JWT_SECRET")
	allowedOrigins := strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000"), ",")

	upgrader := websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			for _, o := range allowedOrigins {
				if strings.TrimSpace(o) == origin {
					return true
				}
			}
			return false
		},
		EnableCompression: false,
	}

	var rdb *redis.Client
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:         addr,
			PoolSize:     50,
			MinIdleConns: 10,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			logger.Error("redis unreachable, running without pub/sub: %v", err)
			rdb = nil
		} else {
			logger.Info("redis connected for ws pub/sub")
		}
	}

	hub := wsgateway.NewHub(rdb)
	go hub.Run()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"healthy","service":"ws-gateway","users":%d,"connections":%d}`,
			hub.ConnectedUsers(), hub.TotalConnections())
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID := authenticateUpgrade(r, jwtSecret)
		if userID == "" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("ws upgrade failed: %v", err)
			return
		}

		client := wsgateway.NewClient(hub, conn, userID)
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,

		// ReadHeaderTimeout protects the upgrade handshake against slowloris.
		// ReadTimeout/WriteTimeout are NOT set: they would kill long-lived
		// WebSocket connections. Gorilla manages per-frame deadlines internally.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,

		// Pre-allocate for high connection count.
		ConnState: func(conn net.Conn, state http.ConnState) {
			if state == http.StateNew {
				_ = conn.(*net.TCPConn).SetKeepAlive(true)
				_ = conn.(*net.TCPConn).SetKeepAlivePeriod(30 * time.Second)
			}
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("ws-gateway started on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error: %v", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down ws-gateway...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	if rdb != nil {
		_ = rdb.Close()
	}
	logger.Info("ws-gateway stopped")
}

// raiseFileLimit attempts to raise the process file descriptor limit
// to support 100K+ concurrent connections.
func raiseFileLimit() {
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		logger.Error("getrlimit failed: %v", err)
		return
	}
	if rlimit.Cur < rlimit.Max {
		rlimit.Cur = rlimit.Max
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
			logger.Error("setrlimit failed: %v", err)
		}
	}
	logger.Info("fd limit: cur=%d max=%d", rlimit.Cur, rlimit.Max)
}

// authenticateUpgrade validates JWT before the WebSocket handshake completes.
// Accepts token via ?token= query param (browser-friendly) or Authorization header.
func authenticateUpgrade(r *http.Request, secret string) string {
	var tokenStr string
	if t := r.URL.Query().Get("token"); t != "" {
		tokenStr = t
	} else if h := r.Header.Get("Authorization"); len(h) > 7 && strings.HasPrefix(h, "Bearer ") {
		tokenStr = h[7:]
	}
	if tokenStr == "" {
		return ""
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return ""
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.UserID == "" || !middleware.IsValidUUID(claims.UserID) {
		return ""
	}
	return claims.UserID
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		logger.Fatal("%s environment variable is required", key)
	}
	return v
}
