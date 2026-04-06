package wsgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/whatsapp-groupe4/internal/logger"
)

const (
	redisChan   = "ws:broadcast"
	numShards   = 64
	shardMask   = numShards - 1
	registerBuf = 4096

	sweepInterval   = 30 * time.Second
	zombieThreshold = 2 * pongWait

	sessionTTL = 5 * time.Minute

	// Rooms with more members than this threshold use parallel fan-out.
	// Below this, sequential send is faster (no goroutine overhead).
	fanOutThreshold = 5000
	fanOutBatch     = 1024
)

// ── Shard types ─────────────────────────────────────────────────────────

type clientShard struct {
	mu      sync.RWMutex
	clients map[string][]*Client
}

type roomShard struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
}

// ── Hub ─────────────────────────────────────────────────────────────────

// Hub maintains active clients and room subscriptions across 64 independent
// shards. It runs a sweeper to kill zombie connections and persists session
// state in Redis so clients can reconnect and restore their subscriptions.
type Hub struct {
	cShards [numShards]clientShard
	rShards [numShards]roomShard

	Register   chan *Client
	Unregister chan *Client

	connCount atomic.Int64
	userCount atomic.Int64

	rdb *redis.Client
	ctx context.Context
}

func NewHub(rdb *redis.Client) *Hub {
	h := &Hub{
		Register:   make(chan *Client, registerBuf),
		Unregister: make(chan *Client, registerBuf),
		rdb:        rdb,
		ctx:        context.Background(),
	}
	for i := range h.cShards {
		h.cShards[i].clients = make(map[string][]*Client, 1024)
	}
	for i := range h.rShards {
		h.rShards[i].rooms = make(map[string]map[*Client]struct{}, 256)
	}
	return h
}

func shard(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32() & shardMask
}

// Run is the main event loop. Call as a goroutine.
func (h *Hub) Run() {
	if h.rdb != nil {
		go h.redisSubscriber()
	}
	go h.sweeper()

	for {
		select {
		case c := <-h.Register:
			h.addClient(c)
		case c := <-h.Unregister:
			h.removeClient(c)
		}
	}
}

func (h *Hub) addClient(c *Client) {
	s := &h.cShards[shard(c.UserID)]
	s.mu.Lock()
	prev := len(s.clients[c.UserID])
	s.clients[c.UserID] = append(s.clients[c.UserID], c)
	s.mu.Unlock()

	h.connCount.Add(1)
	if prev == 0 {
		h.userCount.Add(1)
	}

	restoredRooms := h.RestoreSession(c)
	c.SendWelcome(restoredRooms)
}

func (h *Hub) removeClient(c *Client) {
	s := &h.cShards[shard(c.UserID)]
	s.mu.Lock()
	conns := s.clients[c.UserID]
	for i, cc := range conns {
		if cc == c {
			s.clients[c.UserID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	wasLast := len(s.clients[c.UserID]) == 0
	if wasLast {
		delete(s.clients, c.UserID)
	}
	s.mu.Unlock()

	h.connCount.Add(-1)
	if wasLast {
		h.userCount.Add(-1)
	}

	rooms := c.Rooms()
	for _, chatID := range rooms {
		rs := &h.rShards[shard(chatID)]
		rs.mu.Lock()
		if members, ok := rs.rooms[chatID]; ok {
			delete(members, c)
			if len(members) == 0 {
				delete(rs.rooms, chatID)
			}
		}
		rs.mu.Unlock()
	}

	c.CloseSend()
}

// ── Sweeper ─────────────────────────────────────────────────────────────

// sweeper periodically scans all shards for zombie connections that
// gorilla's ping/pong missed (e.g., half-open TCP connections).
func (h *Hub) sweeper() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for range ticker.C {
		var zombies []*Client
		for i := range h.cShards {
			s := &h.cShards[i]
			s.mu.RLock()
			for _, conns := range s.clients {
				for _, c := range conns {
					if c.IdleDuration() > zombieThreshold {
						zombies = append(zombies, c)
					}
				}
			}
			s.mu.RUnlock()
		}

		for _, c := range zombies {
			logger.Info("sweeper: closing zombie connection user=%s idle=%s", c.UserID, c.IdleDuration())
			c.Conn.Close()
		}

		if len(zombies) > 0 {
			logger.Info("sweeper: cleaned %d zombie connections", len(zombies))
		}
	}
}

// ── Session persistence (Redis) ─────────────────────────────────────────

func sessionKey(userID string) string {
	return fmt.Sprintf("ws:session:%s:rooms", userID)
}

// SaveSession persists a room subscription in Redis so it survives reconnection.
func (h *Hub) SaveSession(userID, chatID string) {
	if h.rdb == nil {
		return
	}
	key := sessionKey(userID)
	h.rdb.SAdd(h.ctx, key, chatID)
	h.rdb.Expire(h.ctx, key, sessionTTL)
}

// RemoveSession removes a room from the user's persisted session.
func (h *Hub) RemoveSession(userID, chatID string) {
	if h.rdb == nil {
		return
	}
	key := sessionKey(userID)
	h.rdb.SRem(h.ctx, key, chatID)
}

// RestoreSession reads the user's saved rooms from Redis, re-subscribes
// the client to each, and refreshes the TTL. Returns the list of restored rooms.
func (h *Hub) RestoreSession(c *Client) []string {
	if h.rdb == nil {
		return nil
	}

	key := sessionKey(c.UserID)
	rooms, err := h.rdb.SMembers(h.ctx, key).Result()
	if err != nil || len(rooms) == 0 {
		return nil
	}

	h.rdb.Expire(h.ctx, key, sessionTTL)

	for _, chatID := range rooms {
		c.mu.Lock()
		c.rooms[chatID] = struct{}{}
		c.mu.Unlock()
		h.Subscribe(c, chatID)
	}

	return rooms
}

// ── Room management ─────────────────────────────────────────────────────

// Subscribe adds a client to a chat room.
func (h *Hub) Subscribe(c *Client, chatID string) {
	rs := &h.rShards[shard(chatID)]
	rs.mu.Lock()
	if rs.rooms[chatID] == nil {
		rs.rooms[chatID] = make(map[*Client]struct{}, 16)
	}
	rs.rooms[chatID][c] = struct{}{}
	rs.mu.Unlock()
}

// UnsubscribeRoom removes a client from a chat room.
func (h *Hub) UnsubscribeRoom(c *Client, chatID string) {
	rs := &h.rShards[shard(chatID)]
	rs.mu.Lock()
	if members, ok := rs.rooms[chatID]; ok {
		delete(members, c)
		if len(members) == 0 {
			delete(rs.rooms, chatID)
		}
	}
	rs.mu.Unlock()
}

// ── Broadcast ───────────────────────────────────────────────────────────

// BroadcastToRoom sends an envelope to every client in a room except
// excludeUser. If Redis is configured, publishes for cross-instance relay.
func (h *Hub) BroadcastToRoom(chatID string, env Envelope, excludeUser string) {
	if h.rdb != nil {
		msg := RedisBroadcast{ChatID: chatID, Envelope: env, ExcludeUser: excludeUser}
		if data, err := json.Marshal(msg); err == nil {
			h.rdb.Publish(h.ctx, redisChan, data)
		}
		return
	}
	h.localBroadcast(chatID, env, excludeUser)
}

// BroadcastToUser delivers an envelope to all connections of a given user.
func (h *Hub) BroadcastToUser(userID string, env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	s := &h.cShards[shard(userID)]
	s.mu.RLock()
	conns := s.clients[userID]
	s.mu.RUnlock()

	for _, c := range conns {
		select {
		case c.Send <- data:
		default:
		}
	}
}

func (h *Hub) localBroadcast(chatID string, env Envelope, excludeUser string) {
	data, err := json.Marshal(env)
	if err != nil {
		return
	}

	rs := &h.rShards[shard(chatID)]
	rs.mu.RLock()
	members := rs.rooms[chatID]
	targets := make([]*Client, 0, len(members))
	for c := range members {
		targets = append(targets, c)
	}
	rs.mu.RUnlock()

	if len(targets) < fanOutThreshold {
		sendToAll(targets, data, excludeUser)
		return
	}

	var wg sync.WaitGroup
	for i := 0; i < len(targets); i += fanOutBatch {
		end := i + fanOutBatch
		if end > len(targets) {
			end = len(targets)
		}
		batch := targets[i:end]
		wg.Add(1)
		go func(slice []*Client) {
			defer wg.Done()
			sendToAll(slice, data, excludeUser)
		}(batch)
	}
	wg.Wait()
}

func sendToAll(clients []*Client, data []byte, excludeUser string) {
	for _, c := range clients {
		if c.UserID == excludeUser {
			continue
		}
		select {
		case c.Send <- data:
		default:
		}
	}
}

// ── Stats ───────────────────────────────────────────────────────────────

// ConnectedUsers returns the number of unique authenticated users (O(1)).
func (h *Hub) ConnectedUsers() int {
	return int(h.userCount.Load())
}

// TotalConnections returns total active WebSocket connections (O(1)).
func (h *Hub) TotalConnections() int {
	return int(h.connCount.Load())
}

// ── Redis subscriber ────────────────────────────────────────────────────

func (h *Hub) redisSubscriber() {
	sub := h.rdb.Subscribe(h.ctx, redisChan)
	defer sub.Close()

	ch := sub.Channel(redis.WithChannelSize(4096))
	for msg := range ch {
		var b RedisBroadcast
		if err := json.Unmarshal([]byte(msg.Payload), &b); err != nil {
			logger.Error("redis ws unmarshal: %v", err)
			continue
		}
		h.localBroadcast(b.ChatID, b.Envelope, b.ExcludeUser)
	}
}
