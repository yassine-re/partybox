import { env } from "$env/dynamic/public";
import type {
  Box,
  Completion,
  Game,
  Mission,
  Player,
  Session,
} from "./types";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(
  path: string,
  token?: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";
  let response: Response;
  try {
    response = await fetch(
      `${(env.PUBLIC_API_URL || "/api").replace(/\/$/, "")}${path}`,
      {
        method: body === undefined ? "GET" : "POST",
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: AbortSignal.timeout(12_000),
      },
    );
  } catch {
    throw new ApiError(
      0,
      "Connexion interrompue. Vérifie ton réseau puis réessaie.",
    );
  }
  const data = await response.json().catch(() => null);
  if (!response.ok)
    throw new ApiError(
      response.status,
      data?.error?.message || "Le serveur est momentanément indisponible.",
    );
  return data as T;
}

export const api = {
  box: (id: string) => request<Box>(`/boxes/${encodeURIComponent(id)}`),
  create: (id: string, name: string, player_name: string) =>
    request<Session>(`/boxes/${encodeURIComponent(id)}/games`, undefined, {
      name,
      player_name,
    }),
  join: (id: string, name: string) =>
    request<Session>(`/games/${id}/join`, undefined, { name }),
  me: (token: string) => request<Player>("/players/me", token),
  game: (id: string, token: string) => request<Game>(`/games/${id}`, token),
  mission: (token: string) =>
    request<{ mission: Mission | null }>("/players/me/mission", token),
  start: (id: string, token: string) =>
    request(`/games/${id}/start`, token, {}),
  end: (id: string, token: string) => request(`/games/${id}/end`, token, {}),
  websocketTicket: (id: string, token: string) =>
    request<{ ticket: string; expires_at: string }>(
      `/games/${id}/ws-ticket`,
      token,
      {},
    ),
  complete: (token: string, assignment_id: string) =>
    request<Completion>("/players/me/mission/complete", token, {
      assignment_id,
    }),
};
