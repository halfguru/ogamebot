import type { WSMessage } from '@ogame-bot/shared';

export interface WSClient {
  connect: () => void;
  disconnect: () => void;
}

export function createWSClient(
  onMessage: (msg: WSMessage) => void,
  onStatusChange?: (connected: boolean) => void,
): WSClient {
  let ws: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let shouldReconnect = true;
  let attempt = 0;
  const maxDelay = 30000;
  const baseDelay = 1000;

  const wsBase = typeof location !== 'undefined'
    ? `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}`
    : 'ws://localhost:3000';

  function getBackoffDelay(): number {
    const delay = Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
    attempt++;
    return delay;
  }

  function resetBackoff() {
    attempt = 0;
  }

  function scheduleReconnect() {
    if (!shouldReconnect) return;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    const delay = getBackoffDelay();
    console.log(`[WS] Reconnecting in ${delay}ms (attempt ${attempt})`);
    reconnectTimer = setTimeout(() => {
      connect();
    }, delay);
  }

  function connect() {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    shouldReconnect = true;

    try {
      ws = new WebSocket(`${wsBase}/ws`);

      ws.onopen = () => {
        resetBackoff();
        onStatusChange?.(true);
      };

      ws.onmessage = (event: MessageEvent) => {
        try {
          const msg = JSON.parse(event.data as string) as WSMessage;
          onMessage(msg);
        } catch (err) {
          console.error('[WS] Failed to parse message:', err);
        }
      };

      ws.onclose = () => {
        onStatusChange?.(false);
        ws = null;
        scheduleReconnect();
      };

      ws.onerror = () => {
        onStatusChange?.(false);
      };
    } catch (err) {
      console.error('[WS] Failed to create WebSocket:', err);
      scheduleReconnect();
    }
  }

  function disconnect() {
    shouldReconnect = false;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (ws) {
      ws.close();
      ws = null;
    }
    onStatusChange?.(false);
  }

  return { connect, disconnect };
}
