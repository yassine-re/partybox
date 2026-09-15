#include "diagnostics.h"

#include <WiFi.h>
#include <Wire.h>
#include <esp_system.h>

#include "board_config.h"

void Diagnostics::begin() {
  Serial.printf("[BOOT] modèle=%s révision=%d cœurs=%d\n", ESP.getChipModel(), ESP.getChipRevision(), ESP.getChipCores());
  Serial.printf("[BOOT] flash=%u octets PSRAM=%u octets reset_reason=%d\n", ESP.getFlashChipSize(), ESP.getPsramSize(), esp_reset_reason());
  Serial.printf("[BOOT] MAC Wi-Fi=%s\n", WiFi.macAddress().c_str());
  Serial.printf("[BOOT] pins S2=%d S3=%d\n", BUTTON_S2_PIN, BUTTON_S3_PIN);
  Serial.printf("[BOOT] bus I2C SDA=%d SCL=%d RGB=0x%02X activation=GPIO%d\n", I2C_SDA_PIN,
                I2C_SCL_PIN, RGB_LED_I2C_ADDRESS, RGB_LED_ENABLE_PIN);
  if (!configuredPin(BUTTON_S2_PIN) || !configuredPin(BUTTON_S3_PIN) ||
      !configuredPin(I2C_SDA_PIN) || !configuredPin(I2C_SCL_PIN) ||
      !configuredPin(RGB_LED_ENABLE_PIN)) {
    Serial.println("[ERROR] pinout incomplet : fonctions hardware concernées désactivées");
  }
  for (int pin : DIAGNOSTIC_CANDIDATE_INPUT_PINS) {
    if (configuredPin(pin)) pinMode(pin, INPUT_PULLUP);
  }
  if (configuredPin(I2C_SDA_PIN) && configuredPin(I2C_SCL_PIN)) {
    Serial.println("[I2C] scan du bus");
    for (uint8_t address = 1; address < 127; ++address) {
      Wire.beginTransmission(address);
      if (Wire.endTransmission() == 0) {
        Serial.printf("[I2C] périphérique détecté à 0x%02X\n", address);
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
}
