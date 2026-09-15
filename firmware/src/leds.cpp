#include "leds.h"

#include <Wire.h>

#include "board_config.h"

namespace {
constexpr uint8_t NCP5623_SHUTDOWN = 0x00;
constexpr uint8_t NCP5623_CURRENT = 0x20;
constexpr uint8_t NCP5623_RED_PWM = 0x40;
constexpr uint8_t NCP5623_GREEN_PWM = 0x60;
constexpr uint8_t NCP5623_BLUE_PWM = 0x80;
}  // namespace

void ReactionLeds::begin() {
  if (!configuredPin(RGB_LED_ENABLE_PIN)) return;

  // Keep D19 unpowered until all three PWM channels have a known off state.
  digitalWrite(RGB_LED_ENABLE_PIN, RGB_LED_ENABLE_ACTIVE_LOW ? HIGH : LOW);
  pinMode(RGB_LED_ENABLE_PIN, OUTPUT);
  initialized_ = send(NCP5623_SHUTDOWN) &&
                 send(NCP5623_CURRENT | (RGB_LED_CURRENT_STEP & 0x1F)) &&
                 send(NCP5623_RED_PWM) && send(NCP5623_GREEN_PWM) &&
                 send(NCP5623_BLUE_PWM);
  if (!initialized_) {
    Serial.println("[RGB] initialisation NCP5623 impossible");
    return;
  }
  digitalWrite(RGB_LED_ENABLE_PIN, RGB_LED_ENABLE_ACTIVE_LOW ? LOW : HIGH);
  Serial.println("[RGB] NCP5623 prêt");
}

bool ReactionLeds::available() const { return initialized_; }

bool ReactionLeds::send(uint8_t command) {
  Wire.beginTransmission(RGB_LED_I2C_ADDRESS);
  Wire.write(command);
  return Wire.endTransmission() == 0;
}

bool ReactionLeds::writeState(bool red, bool green) {
  if (!initialized_) return false;
  if (red == redOn_ && green == greenOn_) return true;
  const bool written =
      send(NCP5623_RED_PWM | (red ? RGB_LED_PWM_ON : 0)) &&
      send(NCP5623_GREEN_PWM | (green ? RGB_LED_PWM_ON : 0));
  if (written) {
    redOn_ = red;
    greenOn_ = green;
  } else {
    Serial.println("[RGB] écriture I2C impossible");
  }
  return written;
}

void ReactionLeds::set(bool red, bool green) {
  if (confirmationUntilMs_ != 0) return;
  writeState(red, green);
}

void ReactionLeds::confirm(unsigned long nowMs) {
  confirmationUntilMs_ = nowMs + 700;
  writeState(false, true);
}

void ReactionLeds::loop(unsigned long nowMs) {
  if (confirmationUntilMs_ != 0 && static_cast<long>(nowMs - confirmationUntilMs_) >= 0) {
    confirmationUntilMs_ = 0;
    writeState(false, false);
  }
}
