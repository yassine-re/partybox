#pragma once

#include <Arduino.h>

class ReactionLeds {
 public:
  void begin();
  bool available() const;
  void set(bool red, bool green);
  void confirm(unsigned long nowMs);
  void loop(unsigned long nowMs);

 private:
  bool send(uint8_t command);
  bool writeState(bool red, bool green);
  bool initialized_ = false;
  bool redOn_ = false;
  bool greenOn_ = false;
  unsigned long confirmationUntilMs_ = 0;
};
