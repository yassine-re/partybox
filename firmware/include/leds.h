#pragma once

#include <Arduino.h>

class ReactionLeds {
 public:
  ReactionLeds(int redPin, int greenPin, bool activeLow);
  void begin();
  void set(bool red, bool green);
  void confirm(unsigned long nowMs);
  void loop(unsigned long nowMs);

 private:
  void write(int pin, bool on);
  int redPin_;
  int greenPin_;
  bool activeLow_;
  unsigned long confirmationUntilMs_ = 0;
};
