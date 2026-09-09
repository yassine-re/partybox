import type {
  Player,
  ReactionChallenge,
  ReactionState,
} from "./api/types";

export function canAssignReaction(
  state: ReactionState | null,
  isHost: boolean,
): boolean {
  return Boolean(
    isHost &&
      state?.enabled &&
      state.device_online &&
      state.challenge?.status === "awaiting_assignment",
  );
}

export function validReactionAssignment(
  challenge: ReactionChallenge,
  s2PlayerId: string,
  s3PlayerId: string,
): boolean {
  if (challenge.kind === "solo") {
    return Boolean(s2PlayerId) !== Boolean(s3PlayerId);
  }
  return Boolean(s2PlayerId && s3PlayerId && s2PlayerId !== s3PlayerId);
}

export function reactionResultText(
  challenge: ReactionChallenge,
  players: Player[],
): string {
  const playerName = (id: string | null, provided: string | null) =>
    provided ?? players.find((player) => player.id === id)?.name ?? "Un joueur";
  if (challenge.false_start_player_id) {
    const starter = playerName(
      challenge.false_start_player_id,
      challenge.false_start_player_name,
    );
    if (challenge.winner_player_id) {
      const winner = playerName(
        challenge.winner_player_id,
        challenge.winner_player_name,
      );
      return `Faux départ de ${starter} ! ${winner} gagne — +${challenge.awarded_points} pts`;
    }
    return `Faux départ de ${starter} — 0 pt`;
  }
  if (!challenge.winner_player_id) return "Temps écoulé — 0 pt";
  const winner = playerName(
    challenge.winner_player_id,
    challenge.winner_player_name,
  );
  const timing = challenge.reaction_ms === null ? "" : ` — ${challenge.reaction_ms} ms`;
  return `${winner} gagne !${timing} — +${challenge.awarded_points} pts`;
}
