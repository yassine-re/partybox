#pragma once

// The Insensia PCB pinout is unknown. Keep every unverified pin disabled (-1).
// PCB references such as S2, S3, D19 or Q7 are NOT GPIO numbers.
constexpr int BUTTON_S2_PIN = -1;
constexpr int BUTTON_S3_PIN = -1;
constexpr int LED_RED_PIN = -1;
constexpr int LED_GREEN_PIN = -1;

constexpr bool BUTTON_ACTIVE_LOW = true;
constexpr bool LED_ACTIVE_LOW = false;
constexpr unsigned long BUTTON_DEBOUNCE_MS = 30;

// Diagnostic mode only touches entries explicitly added here. The -1 sentinel
// is ignored. Never populate the output list without tracing the PCB first.
constexpr int DIAGNOSTIC_CANDIDATE_INPUT_PINS[] = {-1};
constexpr int DIAGNOSTIC_ALLOWED_LED_PINS[] = {-1};
constexpr bool DIAGNOSTIC_ENABLE_LED_TEST = false;

constexpr bool configuredPin(int pin) { return pin >= 0; }
