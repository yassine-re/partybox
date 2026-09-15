#pragma once

#include <Arduino.h>

class PartyBoxWiFi {
 public:
  void begin();
  void loop(unsigned long nowMs);
  bool connected() const;
  bool clockSynchronized() const;

 private:
  unsigned long nextAttemptMs_ = 0;
  unsigned long retryMs_ = 1000;
  bool wasConnected_ = false;
  bool clockSyncStarted_ = false;
  bool clockSynchronized_ = false;
};
