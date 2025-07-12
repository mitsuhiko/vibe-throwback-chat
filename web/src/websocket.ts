import { createSignal, createEffect } from "solid-js";
import type {
  WebSocketRequest,
  WebSocketMessage,
  WebSocketResponse,
  ConnectionState,
} from "./types";

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private reconnectTimer: number | null = null;
  private heartbeatTimer: number | null = null;
  private requestCounter = 0;
  private pendingRequests = new Map<
    string,
    (response: WebSocketResponse) => void
  >();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private baseReconnectDelay = 1000;
  private heartbeatInterval = 25000; // 25 seconds
  private url: string;
  private sessionId: string | null = null;
  private readonly SESSION_STORAGE_KEY = "tbchat_session_id";

  // Signals for reactive state
  private connectionStateSignal = createSignal<ConnectionState>("disconnected");
  private lastErrorSignal = createSignal<string | null>(null);

  // Event handlers
  public onMessage: (message: WebSocketMessage) => void = () => {};
  public onConnectionChange: (state: ConnectionState) => void = () => {};

  constructor(url: string = "/ws") {
    this.url = url;
    this.loadStoredSessionId();

    // React to connection state changes
    createEffect(() => {
      this.onConnectionChange(this.connectionStateSignal[0]());
    });
  }

  public getConnectionState() {
    return this.connectionStateSignal[0]();
  }

  public getLastError() {
    return this.lastErrorSignal[0]();
  }

  public connect(sessionId?: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    // Use provided sessionId or stored one
    if (sessionId) {
      this.sessionId = sessionId;
      this.storeSessionId(sessionId);
    }

    this.connectionStateSignal[1]("connecting");
    this.lastErrorSignal[1](null);

    try {
      // Construct WebSocket URL with session ID if available
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      let wsUrl = `${protocol}//${window.location.host}${this.url}`;

      if (this.sessionId) {
        wsUrl += `?session_id=${encodeURIComponent(this.sessionId)}`;
      }

      this.ws = new WebSocket(wsUrl);
      this.setupEventHandlers();
    } catch (error) {
      console.error("Failed to create WebSocket connection:", error);
      this.lastErrorSignal[1](
        error instanceof Error ? error.message : "Failed to connect",
      );
      this.connectionStateSignal[1]("error");
      this.scheduleReconnect();
    }
  }

  public disconnect(clearSession = false): void {
    this.clearTimers();
    this.reconnectAttempts = 0;

    if (this.ws) {
      this.ws.close(1000, "User initiated disconnect");
      this.ws = null;
    }

    if (clearSession) {
      this.clearStoredSessionId();
      this.sessionId = null;
    }

    this.connectionStateSignal[1]("disconnected");
  }

  public send<T extends WebSocketResponse>(
    request: WebSocketRequest,
  ): Promise<T> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error("WebSocket is not connected"));
        return;
      }

      // Generate unique request ID
      const reqId = `req_${++this.requestCounter}_${Date.now()}`;
      request.req_id = reqId;

      // Store pending request
      this.pendingRequests.set(
        reqId,
        resolve as (response: WebSocketResponse) => void,
      );

      // Set timeout for request
      setTimeout(() => {
        if (this.pendingRequests.has(reqId)) {
          this.pendingRequests.delete(reqId);
          reject(new Error("Request timeout"));
        }
      }, 10000); // 10 second timeout

      try {
        this.ws.send(JSON.stringify(request));
      } catch (error) {
        this.pendingRequests.delete(reqId);
        reject(error);
      }
    });
  }

  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log("WebSocket connected");
      this.connectionStateSignal[1]("connected");
      this.lastErrorSignal[1](null);
      this.reconnectAttempts = 0;
      this.startHeartbeat();
    };

    this.ws.onclose = (event) => {
      console.log("WebSocket closed:", event.code, event.reason);
      this.clearTimers();

      if (event.code === 1000) {
        // Normal closure
        this.connectionStateSignal[1]("disconnected");
      } else {
        // Unexpected closure
        this.connectionStateSignal[1]("error");
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      this.lastErrorSignal[1]("Connection error");
      this.connectionStateSignal[1]("error");
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error, event.data);
      }
    };
  }

  private handleMessage(message: WebSocketMessage): void {
    // Handle responses to pending requests
    if (message.type === "response" && "req_id" in message) {
      const reqId = message.req_id;
      const callback = this.pendingRequests.get(reqId);

      if (callback) {
        this.pendingRequests.delete(reqId);
        callback(message as WebSocketResponse);
        return;
      }
    }

    // Pass all messages (including unmatched responses) to the message handler
    this.onMessage(message);
  }

  private startHeartbeat(): void {
    this.clearHeartbeat();

    this.heartbeatTimer = window.setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send({
          cmd: "heartbeat",
          req_id: "", // Will be set by send()
        }).catch((error) => {
          console.warn("Heartbeat failed:", error);
        });
      }
    }, this.heartbeatInterval);
  }

  private clearHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("Max reconnection attempts reached");
      this.connectionStateSignal[1]("error");
      return;
    }

    this.connectionStateSignal[1]("reconnecting");

    const delay = this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts);
    console.log(
      `Scheduling reconnect in ${delay}ms (attempt ${this.reconnectAttempts + 1})`,
    );

    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, delay);
  }

  private clearTimers(): void {
    this.clearHeartbeat();

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  public destroy(): void {
    this.clearTimers();
    this.pendingRequests.clear();

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  public getSessionId(): string | null {
    return this.sessionId;
  }

  public setSessionId(sessionId: string): void {
    this.sessionId = sessionId;
    this.storeSessionId(sessionId);
  }

  public clearSession(): void {
    this.sessionId = null;
    this.clearStoredSessionId();
  }

  private loadStoredSessionId(): void {
    try {
      const stored = sessionStorage.getItem(this.SESSION_STORAGE_KEY);
      if (stored) {
        this.sessionId = stored;
        console.log("Loaded stored session ID:", stored);
      }
    } catch (error) {
      console.warn("Failed to load session ID from storage:", error);
    }
  }

  private storeSessionId(sessionId: string): void {
    try {
      sessionStorage.setItem(this.SESSION_STORAGE_KEY, sessionId);
      console.log("Stored session ID:", sessionId);
    } catch (error) {
      console.warn("Failed to store session ID:", error);
    }
  }

  private clearStoredSessionId(): void {
    try {
      sessionStorage.removeItem(this.SESSION_STORAGE_KEY);
      console.log("Cleared stored session ID");
    } catch (error) {
      console.warn("Failed to clear session ID from storage:", error);
    }
  }
}

// Create and export a singleton instance
export const wsClient = new WebSocketClient();

// Cleanup on page unload
if (typeof window !== "undefined") {
  window.addEventListener("beforeunload", () => {
    wsClient.destroy();
  });
}
