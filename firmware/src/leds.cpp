#include "leds.h"

#include "board_config.h"

ReactionLeds::ReactionLeds(int redPin, int greenPin, bool activeLow)
    : redPin_(redPin), greenPin_(greenPin), activeLow_(activeLow) {}

void ReactionLeds::begin() {
  if (configuredPin(redPin_)) {
    pinMode(redPin_, OUTPUT);
    write(redPin_, false);
  }
  if (configuredPin(greenPin_)) {
    pinMode(greenPin_, OUTPUT);
    write(greenPin_, false);
  }
}

void ReactionLeds::write(int pin, bool on) {
  if (!configuredPin(pin)) return;
  digitalWrite(pin, (on != activeLow_) ? HIGH : LOW);
}

void ReactionLeds::set(bool red, bool green) {
  if (confirmationUntilMs_ != 0) return;
  write(redPin_, red);
  write(greenPin_, green);
}

void ReactionLeds::confirm(unsigned long nowMs) {
  confirmationUntilMs_ = nowMs + 700;
  write(redPin_, false);
  write(greenPin_, true);
}

void ReactionLeds::loop(unsigned long nowMs) {
  if (confirmationUntilMs_ != 0 && static_cast<long>(nowMs - confirmationUntilMs_) >= 0) {
    confirmationUntilMs_ = 0;
    write(redPin_, false);
    write(greenPin_, false);
  }
}
