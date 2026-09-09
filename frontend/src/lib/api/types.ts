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

export type MissionFeedbackRating = -1 | 0 | 1;

export interface MissionFeedback {
  assignment_id: string;
  rating: MissionFeedbackRating;
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

export type ReactionKind = "solo" | "duel";
export type ReactionStatus =
  | "awaiting_assignment"
  | "awaiting_device"
  | "armed"
  | "resolved"
  | "expired"
  | "cancelled";

export interface ReactionChallenge {
  id: string;
  game_id: string;
  box_id: string;
  kind: ReactionKind;
  status: ReactionStatus;
  button_s2_player_id: string | null;
  button_s2_player_name: string | null;
  button_s3_player_id: string | null;
  button_s3_player_name: string | null;
  delay_ms: number;
  winner_player_id: string | null;
  winner_player_name: string | null;
  false_start_player_id: string | null;
  false_start_player_name: string | null;
  reaction_ms: number | null;
  awarded_points: number;
  scheduled_at: string;
  assigned_at: string | null;
  armed_at: string | null;
  resolved_at: string | null;
  expires_at: string;
}

export interface ReactionState {
  enabled: boolean;
  device_online: boolean;
  challenge: ReactionChallenge | null;
}

export interface ReactionAssignment {
  button_s2_player_id: string | null;
  button_s3_player_id: string | null;
}
