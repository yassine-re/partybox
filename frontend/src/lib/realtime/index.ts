import { env } from "$env/dynamic/public";
import { api } from "$lib/api";

export type RealtimeStatus =
  | "connecting"
  | "connected"
  | "reconnecting"
  | "offline";

export interface RealtimeEvent {
  type:
    | "player_joined"
    | "game_started"
    | "mission_completed"
    | "game_ended";
  game_id: string;
  player_id?: string;
  occurred_at: string;
}

interface RealtimeCallbacks {
  onEvent: (event: RealtimeEvent) => void;
  onStatus: (status: RealtimeStatus) => void;
}

const eventTypes = new Set<RealtimeEvent["type"]>([
  "player_joined",
  "game_started",
  "mission_completed",
  "game_ended",
]);

function websocketURL(gameId: string, ticket: string): string {
  const apiBase = (env.PUBLIC_API_URL || "/api").replace(/\/$/, "");
  const target = new URL(
    `${apiBase}/games/${encodeURIComponent(gameId)}/ws`,
    window.location.origin,
  );
  target.protocol = target.protocol === "https:" ? "wss:" : "ws:";
  target.searchParams.set("ticket", ticket);
  return target.toString();
}

export class GameRealtime {
  private socket: WebSocket | null = null;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private generation = 0;
  private retry = 0;
  private active = false;

  constructor(
    private readonly gameId: string,
    private readonly token: string,
    private readonly callbacks: RealtimeCallbacks,
  ) {}

  start() {
    if (this.active) return;
    this.active = true;
    this.retry = 0;
    const generation = ++this.generation;
    void this.connect(generation, false);
  }

  stop() {
    this.active = false;
    this.generation += 1;
    if (this.retryTimer) clearTimeout(this.retryTimer);
    this.retryTimer = null;
    const socket = this.socket;
    this.socket = null;
    if (socket && socket.readyState < WebSocket.CLOSING) {
      socket.close(1000, "session closed");
    }
    this.callbacks.onStatus("offline");
  }

  reconnectNow() {
    if (!this.active || this.socket?.readyState === WebSocket.OPEN) return;
    if (this.retryTimer) clearTimeout(this.retryTimer);
    this.retryTimer = null;
    const previous = this.socket;
    this.socket = null;
    if (previous && previous.readyState < WebSocket.CLOSING) previous.close();
    const generation = ++this.generation;
    void this.connect(generation, this.retry > 0);
  }

  private async connect(generation: number, reconnecting: boolean) {
    if (!this.active || generation !== this.generation) return;
    this.callbacks.onStatus(reconnecting ? "reconnecting" : "connecting");

    let ticket: string;
    try {
      ticket = (await api.websocketTicket(this.gameId, this.token)).ticket;
    } catch {
      if (this.active && generation === this.generation) this.scheduleRetry();
      return;
    }
    if (!this.active || generation !== this.generation) return;

    const socket = new WebSocket(websocketURL(this.gameId, ticket));
    this.socket = socket;
    const openingTimeout = setTimeout(() => socket.close(), 10_000);

    socket.onopen = () => {
      clearTimeout(openingTimeout);
      if (!this.active || generation !== this.generation) {
        socket.close();
        return;
      }
      this.retry = 0;
      this.callbacks.onStatus("connected");
    };
    socket.onmessage = (message) => {
      if (!this.active || generation !== this.generation) return;
      try {
        const event = JSON.parse(String(message.data)) as RealtimeEvent;
        if (event.game_id === this.gameId && eventTypes.has(event.type)) {
          this.callbacks.onEvent(event);
        }
      } catch {
        // Ignore malformed notifications; REST state is unaffected.
      }
    };
    socket.onerror = () => socket.close();
    socket.onclose = () => {
      clearTimeout(openingTimeout);
      if (this.socket !== socket) return;
      this.socket = null;
      if (this.active && generation === this.generation) this.scheduleRetry();
    };
  }

  private scheduleRetry() {
    if (!this.active || this.retryTimer) return;
    this.callbacks.onStatus("reconnecting");
    const delay = Math.min(1000 * 2 ** this.retry, 30_000);
    this.retry += 1;
    const generation = this.generation;
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      void this.connect(generation, true);
    }, delay);
  }
}
