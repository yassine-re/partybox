#pragma once

#include <Arduino.h>

class DeviceIdentity {
 public:
  void begin();
  String nextEventId();

 private:
  String bootId_;
  uint32_t counter_ = 0;
};
