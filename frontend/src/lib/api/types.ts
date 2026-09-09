export type GameMode = "secret_missions" | "treasure_hunt" | "chaos";

export type ChaosEventType =
  | "double_trouble"
  | "bounty"
  | "mission_shuffle";

export interface ChaosState {
  active: boolean;
  event: {
    type: ChaosEventType;
    remaining_uses: number;
    target_player_id: string | null;
    target_player_name: string | null;
    bonus?: number;
  } | null;
  progress: {
    completed: number;
    trigger_at: number;
  };
  sequence: number;
}

export interface Player {
  id: string;
  game_id: string;
  name: string;
  score: number;
  is_host: boolean;
  completed_missions: number;
}
export interface Game {
  id: string;
  box_id: string;
  name: string;
  mode: GameMode;
  status: "lobby" | "playing" | "ended";
  players?: Player[];
  created_at: string;
  started_at: string | null;
  ended_at: string | null;
}
export interface Box {
  id: string;
  name: string;
  active_game: Game | null;
}
export interface Mission {
  id: string;
  mission_id: number;
  text: string;
  points: number;
  category: string;
  difficulty: number;
}
export interface Session {
  token: string;
  player: Player;
  game: Game;
}
export interface SavedSession {
  token: string;
  gameId: string;
}
export interface Completion {
  awarded_points: number;
  already_completed: boolean;
  player: Player;
  mission: Mission | null;
}

export interface MissionProof {
  id: string;
  assignment_id: string;
  verdict: "valid" | "invalid" | "uncertain";
  confidence: number;
  reason: string;
  created_at: string;
}

export interface ProofStatus {
  available: boolean;
  assignment_id?: string;
  attempts: number;
  max_attempts: number;
  remaining_attempts: number;
  accepted: boolean;
}

export interface ProofResponse {
  proof: MissionProof;
  completion?: Completion;
}

export interface AIStatus {
  available: boolean;
  count: number;
  remaining_generations: number;
}

export interface AIGenerationOptions {
  vibe: "chill" | "fun" | "chaos";
  intensity: number;
  context?: string;
  count?: number;
}
