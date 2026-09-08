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
  mode: "secret_missions";
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
