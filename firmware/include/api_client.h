#pragma once

#include <Arduino.h>

#include "reaction_game.h"

class ApiClient {
 public:
  bool heartbeat(uint64_t uptimeMs, int wifiRssi);
  bool pollCommand(ReactionCommand& command);
  bool acknowledge(const String& commandId);
  bool submitReaction(const ReactionOutcome& outcome, const String& eventId);

 private:
  int request(const char* method, const String& path, const String& body, String& response);
};
