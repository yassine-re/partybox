#include "reaction_game.h"

#include <algorithm>

bool ReactionGame::arm(const ReactionCommand& command, int64_t nowUs, bool s2Held, bool s3Held) {
  if (state_ != ReactionState::IDLE || command.commandId.empty() || command.challengeId.empty() ||
      command.delayMs < 2000 || command.delayMs > 6000 || (!command.s2Enabled && !command.s3Enabled)) {
    return false;
  }
  command_ = command;
  outcome_ = ReactionOutcome{};
  outcome_.commandId = command.commandId;
  outcome_.challengeId = command.challengeId;
  greenAtUs_ = nowUs + static_cast<int64_t>(command.delayMs) * 1000;
  ignoreS2UntilRelease_ = command.s2Enabled && s2Held;
  ignoreS3UntilRelease_ = command.s3Enabled && s3Held;
  reportAttempts_ = 0;
  nextReportAtMs_ = 0;
  state_ = ReactionState::ARMED_RED;
  return true;
}

void ReactionGame::update(int64_t nowUs, bool s2PressedEdge, bool s3PressedEdge, bool s2Held, bool s3Held) {
  if (ignoreS2UntilRelease_ && !s2Held) ignoreS2UntilRelease_ = false;
  if (ignoreS3UntilRelease_ && !s3Held) ignoreS3UntilRelease_ = false;
  const bool s2 = command_.s2Enabled && s2PressedEdge && !ignoreS2UntilRelease_;
  const bool s3 = command_.s3Enabled && s3PressedEdge && !ignoreS3UntilRelease_;

  if (state_ == ReactionState::ARMED_RED) {
    if (s2 || s3) {
      resolveButton(s2 ? "S2" : "S3", nowUs, true);
      return;
    }
    if (nowUs >= greenAtUs_) {
      startedAtUs_ = nowUs;
      state_ = ReactionState::GREEN_ACTIVE;
    }
  }
  if (state_ == ReactionState::GREEN_ACTIVE) {
    if (s2 || s3) {
      resolveButton(s2 ? "S2" : "S3", nowUs, false);
    } else if (nowUs - startedAtUs_ >= kResponseTimeoutUs) {
      resolveTimeout();
    }
  }
}

void ReactionGame::resolveButton(const char* button, int64_t nowUs, bool falseStart) {
  outcome_.button = button;
  outcome_.falseStart = falseStart;
  outcome_.reactionMs = falseStart ? -1 : static_cast<int32_t>((nowUs - startedAtUs_) / 1000);
  state_ = ReactionState::REPORT_PENDING;
}

void ReactionGame::resolveTimeout() {
  outcome_.timeout = true;
  outcome_.reactionMs = -1;
  state_ = ReactionState::REPORT_PENDING;
}

bool ReactionGame::shouldReport(uint32_t nowMs) const {
  return state_ == ReactionState::REPORT_PENDING &&
         (nextReportAtMs_ == 0 || static_cast<int32_t>(nowMs - nextReportAtMs_) >= 0);
}

void ReactionGame::reportAttempt(uint32_t nowMs, bool success) {
  if (state_ != ReactionState::REPORT_PENDING) return;
  if (success) {
    state_ = ReactionState::IDLE;
    return;
  }
  reportAttempts_ = std::min<uint8_t>(reportAttempts_ + 1, 5);
  const uint32_t delay = std::min<uint32_t>(1000U << (reportAttempts_ - 1), 30000U);
  nextReportAtMs_ = nowMs + delay;
}
