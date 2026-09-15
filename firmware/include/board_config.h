#pragma once

#include <stdint.h>

// Confirmed Insensia vB.1 mappings, traced with the board fully unpowered.
constexpr int BUTTON_S2_PIN = 16;
constexpr int BUTTON_S3_PIN = 15;
constexpr int I2C_SCL_PIN = 2;
constexpr int I2C_SDA_PIN = 3;
constexpr int RGB_LED_ENABLE_PIN = 33;
constexpr uint8_t RGB_LED_I2C_ADDRESS = 0x38;
constexpr uint8_t RGB_LED_CURRENT_STEP = 4;  // About 1 mA with the reference design.
constexpr uint8_t RGB_LED_PWM_ON = 31;

constexpr bool BUTTON_ACTIVE_LOW = true;
constexpr bool RGB_LED_ENABLE_ACTIVE_LOW = true;
constexpr unsigned long BUTTON_DEBOUNCE_MS = 30;

// Diagnostic mode only touches entries explicitly added here. The -1 sentinel
// is ignored. Never populate the output list without tracing the PCB first.
constexpr int DIAGNOSTIC_CANDIDATE_INPUT_PINS[] = {-1};

constexpr bool configuredPin(int pin) { return pin >= 0; }
