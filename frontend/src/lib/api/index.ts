import { env } from "$env/dynamic/public";
import type {
  AIGenerationOptions,
  AIStatus,
  Box,
  ChaosState,
  Completion,
  Game,
  GameMode,
  Mission,
  Player,
  ProofResponse,
  ProofStatus,
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
  timeoutMs = 12_000,
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
        signal: AbortSignal.timeout(timeoutMs),
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
  missionProofStatus: (token: string) => request<ProofStatus>("/players/me/mission/proof", token),
  submitMissionProof: async (token: string, assignmentId: string, image: Blob): Promise<ProofResponse> => {
    const body = new FormData();
    body.append("assignment_id", assignmentId);
    body.append("image", image, "proof.jpg");
    let response: Response;
    try {
      response = await fetch(`${(env.PUBLIC_API_URL || "/api").replace(/\/$/, "")}/players/me/mission/proof`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body,
        signal: AbortSignal.timeout(70_000),
      });
    } catch {
      throw new ApiError(0, "Envoi interrompu. Réessaie : une photo déjà acceptée ne donnera pas de points en double.");
    }
    const data = await response.json().catch(() => null);
    if (!response.ok) throw new ApiError(response.status, data?.error?.message || "Impossible d’analyser cette photo.");
    return data as ProofResponse;
  },
  box: (id: string) => request<Box>(`/boxes/${encodeURIComponent(id)}`),
  create: (id: string, name: string, player_name: string, mode: GameMode) =>
    request<Session>(`/boxes/${encodeURIComponent(id)}/games`, undefined, {
      name,
      player_name,
      mode,
    }),
  join: (id: string, name: string) =>
    request<Session>(`/games/${id}/join`, undefined, { name }),
  me: (token: string) => request<Player>("/players/me", token),
  game: (id: string, token: string) => request<Game>(`/games/${id}`, token),
  chaos: (id: string, token: string) =>
    request<ChaosState>(`/games/${id}/chaos`, token),
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
  aiGenerateMissions: (
    id: string,
    token: string,
    options: AIGenerationOptions,
  ) =>
    request<{ generated: number }>(
      `/games/${id}/ai-missions/generate`,
      token,
      options,
      50_000,
    ),
  aiMissionsStatus: (id: string, token: string) =>
    request<AIStatus>(`/games/${id}/ai-missions/status`, token),
};
