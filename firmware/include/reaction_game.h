#pragma once

#include <cstdint>
#include <string>

enum class ReactionState { IDLE, ARMED_RED, GREEN_ACTIVE, REPORT_PENDING };

struct ReactionCommand {
  std::string commandId;
  std::string challengeId;
  bool duel = false;
  bool s2Enabled = false;
  bool s3Enabled = false;
  uint32_t delayMs = 0;
};

struct ReactionOutcome {
  std::string commandId;
  std::string challengeId;
  std::string button;
  int32_t reactionMs = -1;
  bool falseStart = false;
  bool timeout = false;
};

class ReactionGame {
 public:
  bool arm(const ReactionCommand& command, int64_t nowUs, bool s2Held, bool s3Held);
  void update(int64_t nowUs, bool s2PressedEdge, bool s3PressedEdge, bool s2Held, bool s3Held);
  ReactionState state() const { return state_; }
  const ReactionOutcome& outcome() const { return outcome_; }
  bool redOn() const { return state_ == ReactionState::ARMED_RED; }
  bool greenOn() const { return state_ == ReactionState::GREEN_ACTIVE; }
  bool shouldReport(uint32_t nowMs) const;
  void reportAttempt(uint32_t nowMs, bool success);

 private:
  static constexpr int64_t kResponseTimeoutUs = 5000000;
  void resolveButton(const char* button, int64_t nowUs, bool falseStart);
  void resolveTimeout();

  ReactionState state_ = ReactionState::IDLE;
  ReactionCommand command_{};
  ReactionOutcome outcome_{};
  int64_t greenAtUs_ = 0;
  int64_t startedAtUs_ = 0;
  bool ignoreS2UntilRelease_ = false;
  bool ignoreS3UntilRelease_ = false;
  uint32_t nextReportAtMs_ = 0;
  uint8_t reportAttempts_ = 0;
};
