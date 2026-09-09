#pragma once

#include <cstdint>

class DebouncedButton {
 public:
  DebouncedButton(bool activeLow, uint32_t debounceMs);
  bool update(bool rawHigh, uint32_t nowMs);
  bool pressed() const;

 private:
  bool activeLow_;
  uint32_t debounceMs_;
  bool initialized_ = false;
  bool candidateHigh_ = true;
  bool stableHigh_ = true;
  uint32_t candidateSinceMs_ = 0;
};
