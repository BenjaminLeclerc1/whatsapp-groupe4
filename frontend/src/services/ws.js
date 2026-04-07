/**
 * WebSocket client with automatic reconnection and heartbeat.
 *
 * Usage:
 *   import { createWSClient } from '../services/ws';
 *
 *   const ws = createWSClient({ getToken: () => localStorage.getItem('token') });
 *   ws.on('new_message', (data) => { ... });
 *   ws.on('typing_update', (data) => { ... });
 *   ws.on('connected', () => { ... });
 *   ws.on('disconnected', () => { ... });
 *   ws.subscribe('chat-uuid');
 *   ws.sendMessage('chat-uuid', 'Hello!');
 *   ws.destroy();
 */

const DEFAULT_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';

const MIN_RECONNECT_DELAY = 500;
const MAX_RECONNECT_DELAY = 30000;
const HEARTBEAT_INTERVAL = 25000;
const HEARTBEAT_TIMEOUT = 10000;

export function createWSClient(opts = {}) {
  const url = opts.url || DEFAULT_URL;
  const getToken = opts.getToken || (() => localStorage.getItem('token'));

  let socket = null;
  let reconnectTimer = null;
  let heartbeatTimer = null;
  let heartbeatTimeoutTimer = null;
  let reconnectDelay = MIN_RECONNECT_DELAY;
  let destroyed = false;
  let subscribedRooms = new Set();

  const listeners = {};

  function on(event, fn) {
    if (!listeners[event]) listeners[event] = [];
    listeners[event].push(fn);
  }

  function off(event, fn) {
    if (!listeners[event]) return;
    listeners[event] = listeners[event].filter((f) => f !== fn);
  }

  function emit(event, data) {
    if (!listeners[event]) return;
    for (const fn of listeners[event]) {
      try { fn(data); } catch (e) { console.error('ws listener error:', e); }
    }
  }

  function connect() {
    if (destroyed) return;

    const token = getToken();
    if (!token) {
      scheduleReconnect();
      return;
    }

    const wsUrl = `${url}?token=${encodeURIComponent(token)}`;
    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
      reconnectDelay = MIN_RECONNECT_DELAY;
      emit('connected');
      startHeartbeat();
    };

    socket.onclose = (e) => {
      stopHeartbeat();
      emit('disconnected', { code: e.code, reason: e.reason });
      if (!destroyed) scheduleReconnect();
    };

    socket.onerror = () => {
      // onclose will fire after this
    };

    socket.onmessage = (e) => {
      const lines = e.data.split('\n');
      for (const line of lines) {
        if (!line.trim()) continue;
        try {
          const msg = JSON.parse(line);
          handleMessage(msg);
        } catch (err) {
          console.error('ws parse error:', err);
        }
      }
    };
  }

  function handleMessage(msg) {
    resetHeartbeatTimeout();

    switch (msg.type) {
      case 'welcome':
        if (msg.restored_rooms && msg.restored_rooms.length > 0) {
          for (const room of msg.restored_rooms) {
            subscribedRooms.add(room);
          }
          emit('session_restored', msg.restored_rooms);
        }
        resubscribeRooms();
        emit('welcome', msg);
        break;

      case 'pong':
        break;

      case 'subscribed':
        subscribedRooms.add(msg.chat_id);
        emit('subscribed', msg);
        break;

      case 'unsubscribed':
        subscribedRooms.delete(msg.chat_id);
        emit('unsubscribed', msg);
        break;

      case 'error':
        emit('error', msg);
        break;

      default:
        emit(msg.type, msg);
        break;
    }
  }

  function resubscribeRooms() {
    for (const chatID of subscribedRooms) {
      send({ type: 'subscribe', chat_id: chatID });
    }
  }

  function send(obj) {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(obj));
    }
  }

  // ── Heartbeat ──────────────────────────────────────────────────────

  function startHeartbeat() {
    stopHeartbeat();
    heartbeatTimer = setInterval(() => {
      send({ type: 'ping' });
      heartbeatTimeoutTimer = setTimeout(() => {
        console.warn('ws heartbeat timeout — forcing reconnect');
        if (socket) socket.close();
      }, HEARTBEAT_TIMEOUT);
    }, HEARTBEAT_INTERVAL);
  }

  function resetHeartbeatTimeout() {
    if (heartbeatTimeoutTimer) {
      clearTimeout(heartbeatTimeoutTimer);
      heartbeatTimeoutTimer = null;
    }
  }

  function stopHeartbeat() {
    if (heartbeatTimer) { clearInterval(heartbeatTimer); heartbeatTimer = null; }
    resetHeartbeatTimeout();
  }

  // ── Reconnection with exponential backoff ──────────────────────────

  function scheduleReconnect() {
    if (destroyed) return;
    const jitter = Math.random() * 0.3 * reconnectDelay;
    const delay = Math.min(reconnectDelay + jitter, MAX_RECONNECT_DELAY);
    emit('reconnecting', { delay: Math.round(delay) });
    reconnectTimer = setTimeout(() => {
      reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY);
      connect();
    }, delay);
  }

  // ── Public API ────────────────────────────────────────────────────

  function subscribe(chatID) {
    subscribedRooms.add(chatID);
    send({ type: 'subscribe', chat_id: chatID });
  }

  function unsubscribe(chatID) {
    subscribedRooms.delete(chatID);
    send({ type: 'unsubscribe', chat_id: chatID });
  }

  function sendMessage(chatID, content) {
    send({ type: 'message', chat_id: chatID, content });
  }

  function sendTyping(chatID, typing) {
    send({ type: 'typing', chat_id: chatID, typing });
  }

  function isConnected() {
    return socket && socket.readyState === WebSocket.OPEN;
  }

  function destroy() {
    destroyed = true;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    stopHeartbeat();
    if (socket) {
      socket.onclose = null;
      socket.close();
    }
    socket = null;
  }

  connect();

  return {
    on,
    off,
    subscribe,
    unsubscribe,
    sendMessage,
    sendTyping,
    isConnected,
    destroy,
  };
}
