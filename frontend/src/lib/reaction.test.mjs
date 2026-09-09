import assert from "node:assert/strict";
import test from "node:test";

import {
  canAssignReaction,
  reactionResultText,
  validReactionAssignment,
} from "./reaction.ts";

const challenge = {
  id: "challenge",
  game_id: "game",
  box_id: "PB001",
  kind: "duel",
  status: "awaiting_assignment",
  button_s2_player_id: null,
  button_s2_player_name: null,
  button_s3_player_id: null,
  button_s3_player_name: null,
  winner_player_id: null,
  winner_player_name: null,
  false_start_player_id: null,
  false_start_player_name: null,
  reaction_ms: null,
  awarded_points: 0,
};

test("reaction assignment is host-only and requires an online device", () => {
  const state = { enabled: true, device_online: true, challenge };
  assert.equal(canAssignReaction(state, true), true);
  assert.equal(canAssignReaction(state, false), false);
  assert.equal(canAssignReaction({ ...state, device_online: false }, true), false);
});

test("maps solo and duel players to physical buttons", () => {
  assert.equal(validReactionAssignment({ ...challenge, kind: "solo" }, "p1", ""), true);
  assert.equal(validReactionAssignment({ ...challenge, kind: "solo" }, "p1", "p2"), false);
  assert.equal(validReactionAssignment(challenge, "p1", "p2"), true);
  assert.equal(validReactionAssignment(challenge, "p1", "p1"), false);
});

test("formats wins and false starts from REST state", () => {
  const players = [
    { id: "p1", name: "Alice" },
    { id: "p2", name: "Bob" },
  ];
  assert.equal(
    reactionResultText({ ...challenge, winner_player_id: "p1", reaction_ms: 243, awarded_points: 100 }, players),
    "Alice gagne ! — 243 ms — +100 pts",
  );
  assert.equal(
    reactionResultText({ ...challenge, false_start_player_id: "p1", winner_player_id: "p2", awarded_points: 100 }, players),
    "Faux départ de Alice ! Bob gagne — +100 pts",
  );
});
