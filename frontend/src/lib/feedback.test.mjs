import assert from "node:assert/strict";
import test from "node:test";

import {
  MISSION_FEEDBACK_FREQUENCY,
  shouldRequestMissionFeedback,
} from "./feedback.ts";

test("requests optional feedback every third successful completion", () => {
  assert.equal(MISSION_FEEDBACK_FREQUENCY, 3);
  assert.equal(shouldRequestMissionFeedback(1, false), false);
  assert.equal(shouldRequestMissionFeedback(2, false), false);
  assert.equal(shouldRequestMissionFeedback(3, false), true);
  assert.equal(shouldRequestMissionFeedback(4, false), false);
  assert.equal(shouldRequestMissionFeedback(6, false), true);
});

test("does not request feedback for an idempotent completion retry", () => {
  assert.equal(shouldRequestMissionFeedback(3, true), false);
  assert.equal(shouldRequestMissionFeedback(0, false), false);
});
