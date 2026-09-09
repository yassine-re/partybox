#pragma once

#include <Arduino.h>

class Diagnostics {
 public:
  void begin();
  void loop(unsigned long nowMs);

 private:
  unsigned long nextLogMs_ = 0;
  bool ledState_ = false;
};
