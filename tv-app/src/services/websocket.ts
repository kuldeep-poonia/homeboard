import { CONFIG } from '../config/constants';

export type LiveEventHandler = (eventType: string, payload: unknown) => void;

export class TVWebSocketClient {
  private ws: WebSocket | null = null;
  private isDestroyed = false;
  private reconnectAttempts = 0;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private boardId: string;
  private tvSecret: string;
  private onEvent: LiveEventHandler;
  private onStatusChange: (status: 'connected' | 'reconnecting' | 'disconnected') => void;

  constructor(
    boardId: string,
    tvSecret: string,
    onEvent: LiveEventHandler,
    onStatusChange: (status: 'connected' | 'reconnecting' | 'disconnected') => void
  ) {
    this.boardId = boardId;
    this.tvSecret = tvSecret;
    this.onEvent = onEvent;
    this.onStatusChange = onStatusChange;
  }

  public connect() {
    if (this.isDestroyed) return;

    try {
      const baseUrl = CONFIG.API_BASE_URL.replace(/^http/, 'ws');
      const wsUrl = `${baseUrl}/v1/boards/${this.boardId}/ws?tv_secret=${encodeURIComponent(this.tvSecret)}`;

      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.onStatusChange('connected');
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data && data.type) {
            this.onEvent(data.type, data.payload);
          }
        } catch (err) {
          // ignore malformed live event
        }
      };

      this.ws.onerror = () => {
        this.onStatusChange('reconnecting');
      };

      this.ws.onclose = () => {
        if (!this.isDestroyed) {
          this.scheduleReconnect();
        }
      };
    } catch (e) {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    this.onStatusChange('reconnecting');
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);

    // Exponential backoff with jitter (max 15s)
    const baseDelay = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), 15000);
    const jitter = Math.random() * 1000;
    const delay = baseDelay + jitter;
    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  public disconnect() {
    this.isDestroyed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.onStatusChange('disconnected');
  }
}
