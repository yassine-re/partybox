#include "button.h"

DebouncedButton::DebouncedButton(bool activeLow, uint32_t debounceMs)
    : activeLow_(activeLow), debounceMs_(debounceMs) {}

bool DebouncedButton::update(bool rawHigh, uint32_t nowMs) {
  if (!initialized_) {
    initialized_ = true;
    candidateHigh_ = stableHigh_ = rawHigh;
    candidateSinceMs_ = nowMs;
    return false;
  }
  if (rawHigh != candidateHigh_) {
    candidateHigh_ = rawHigh;
    candidateSinceMs_ = nowMs;
    return false;
  }
  if (candidateHigh_ == stableHigh_ || nowMs - candidateSinceMs_ < debounceMs_) {
    return false;
  }
  bool wasPressed = pressed();
  stableHigh_ = candidateHigh_;
  return !wasPressed && pressed();
}

bool DebouncedButton::pressed() const {
  return activeLow_ ? !stableHigh_ : stableHigh_;
}
