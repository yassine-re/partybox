#include "diagnostics.h"

#include <WiFi.h>
#include <esp_system.h>

#include "board_config.h"

void Diagnostics::begin() {
  Serial.printf("[BOOT] modèle=%s révision=%d cœurs=%d\n", ESP.getChipModel(), ESP.getChipRevision(), ESP.getChipCores());
  Serial.printf("[BOOT] flash=%u octets PSRAM=%u octets reset_reason=%d\n", ESP.getFlashChipSize(), ESP.getPsramSize(), esp_reset_reason());
  Serial.printf("[BOOT] MAC Wi-Fi=%s\n", WiFi.macAddress().c_str());
  Serial.printf("[BOOT] pins S2=%d S3=%d LED rouge=%d LED verte=%d\n", BUTTON_S2_PIN, BUTTON_S3_PIN, LED_RED_PIN, LED_GREEN_PIN);
  if (!configuredPin(BUTTON_S2_PIN) || !configuredPin(BUTTON_S3_PIN) ||
      !configuredPin(LED_RED_PIN) || !configuredPin(LED_GREEN_PIN)) {
    Serial.println("[ERROR] pinout incomplet : fonctions hardware concernées désactivées");
  }
  for (int pin : DIAGNOSTIC_CANDIDATE_INPUT_PINS) {
    if (configuredPin(pin)) pinMode(pin, INPUT_PULLUP);
  }
  if (DIAGNOSTIC_ENABLE_LED_TEST) {
    for (int pin : DIAGNOSTIC_ALLOWED_LED_PINS) {
      if (configuredPin(pin)) {
        pinMode(pin, OUTPUT);
        digitalWrite(pin, LED_ACTIVE_LOW ? HIGH : LOW);
      }
    }
  }
}

void Diagnostics::loop(unsigned long nowMs) {
  if (static_cast<long>(nowMs - nextLogMs_) < 0) return;
  nextLogMs_ = nowMs + 1000;
  for (int pin : DIAGNOSTIC_CANDIDATE_INPUT_PINS) {
    if (configuredPin(pin)) Serial.printf("[BUTTON] candidat GPIO%d=%d\n", pin, digitalRead(pin));
  }
  if (DIAGNOSTIC_ENABLE_LED_TEST) {
    ledState_ = !ledState_;
    for (int pin : DIAGNOSTIC_ALLOWED_LED_PINS) {
      if (configuredPin(pin)) digitalWrite(pin, (ledState_ != LED_ACTIVE_LOW) ? HIGH : LOW);
    }
  }
}
