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

  const wsBase = typeof location !== 'undefined'
    ? `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}`
    : 'ws://localhost:3000';

  function scheduleReconnect() {
    if (!shouldReconnect) return;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(() => {
      connect();
    }, 3000);
  }

  function connect() {
    if (ws) {
      ws.close();
      ws = null;
    }

    shouldReconnect = true;

    try {
      ws = new WebSocket(`${wsBase}/ws`);

      ws.onopen = () => {
        console.log('[WS] Connected');
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
        console.log('[WS] Disconnected');
        onStatusChange?.(false);
        scheduleReconnect();
      };

      ws.onerror = (err) => {
        console.error('[WS] Error:', err);
        onStatusChange?.(false);
        // onclose will fire after onerror, which schedules reconnect
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
