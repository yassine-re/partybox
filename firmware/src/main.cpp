#include <Arduino.h>
#include <WiFi.h>
#include <esp_timer.h>

#include "api_client.h"
#include "board_config.h"
#include "button.h"
#include "config.h"
#include "device_identity.h"
#include "diagnostics.h"
#include "leds.h"
#include "reaction_game.h"
#include "wifi_manager.h"

PartyBoxWiFi partyboxWiFi;
ApiClient api;
DeviceIdentity identity;
Diagnostics diagnostics;
ReactionLeds leds(LED_RED_PIN, LED_GREEN_PIN, LED_ACTIVE_LOW);
DebouncedButton buttonS2(BUTTON_ACTIVE_LOW, BUTTON_DEBOUNCE_MS);
DebouncedButton buttonS3(BUTTON_ACTIVE_LOW, BUTTON_DEBOUNCE_MS);
ReactionGame reaction;

unsigned long nextHeartbeatMs = 0;
unsigned long nextPollMs = 0;
String pendingEventId;
ReactionState previousState = ReactionState::IDLE;

bool rawButtonLevel(int pin) {
  if (!configuredPin(pin)) return BUTTON_ACTIVE_LOW ? HIGH : LOW;
  return digitalRead(pin);
}

bool commandHardwareAvailable(const ReactionCommand& command) {
  return (!command.s2Enabled || configuredPin(BUTTON_S2_PIN)) &&
         (!command.s3Enabled || configuredPin(BUTTON_S3_PIN)) &&
         configuredPin(LED_RED_PIN) && configuredPin(LED_GREEN_PIN);
}

void setup() {
  Serial.begin(115200);
  delay(150);
  Serial.printf("[BOOT] PartyBox firmware %s (%s)\n", PARTYBOX_FIRMWARE_VERSION,
                PARTYBOX_HAS_LOCAL_SECRETS ? "configuration locale" : "configuration exemple");
  diagnostics.begin();
  if (configuredPin(BUTTON_S2_PIN)) pinMode(BUTTON_S2_PIN, BUTTON_ACTIVE_LOW ? INPUT_PULLUP : INPUT_PULLDOWN);
  if (configuredPin(BUTTON_S3_PIN)) pinMode(BUTTON_S3_PIN, BUTTON_ACTIVE_LOW ? INPUT_PULLUP : INPUT_PULLDOWN);
  leds.begin();
  identity.begin();
#if PARTYBOX_NETWORK_ENABLED
  partyboxWiFi.begin();
#else
  Serial.println("[BOOT] build série/diagnostic : réseau désactivé");
#endif
}

void loop() {
  const unsigned long nowMs = millis();
  const bool s2Edge = buttonS2.update(rawButtonLevel(BUTTON_S2_PIN), nowMs);
  const bool s3Edge = buttonS3.update(rawButtonLevel(BUTTON_S3_PIN), nowMs);
  reaction.update(esp_timer_get_time(), s2Edge, s3Edge, buttonS2.pressed(), buttonS3.pressed());
  leds.set(reaction.redOn(), reaction.greenOn());
  leds.loop(nowMs);
  diagnostics.loop(nowMs);
#if PARTYBOX_NETWORK_ENABLED
  partyboxWiFi.loop(nowMs);

  if (reaction.state() != previousState) {
    Serial.printf("[REACTION] état %d -> %d\n", static_cast<int>(previousState), static_cast<int>(reaction.state()));
    previousState = reaction.state();
  }

  if (partyboxWiFi.connected() && reaction.shouldReport(nowMs)) {
    if (pendingEventId.isEmpty()) pendingEventId = identity.nextEventId();
    const bool delivered = api.submitReaction(reaction.outcome(), pendingEventId);
    reaction.reportAttempt(nowMs, delivered);
    if (delivered) {
      pendingEventId = "";
      leds.confirm(nowMs);
      Serial.println("[REACTION] résultat confirmé par le serveur");
    }
  }

  // Network calls are deliberately forbidden while red/green timing is active.
  if (partyboxWiFi.connected() && reaction.state() == ReactionState::IDLE) {
    if (static_cast<long>(nowMs - nextHeartbeatMs) >= 0) {
      api.heartbeat(nowMs, WiFi.RSSI());
      nextHeartbeatMs = nowMs + HEARTBEAT_INTERVAL_MS;
    }
    if (static_cast<long>(nowMs - nextPollMs) >= 0) {
      ReactionCommand command;
      if (api.pollCommand(command)) {
        Serial.printf("[COMMAND] reaction_arm %s\n", command.commandId.c_str());
        if (!commandHardwareAvailable(command)) {
          Serial.println("[ERROR] commande non armée : GPIO requis non configuré");
        } else if (api.acknowledge(command.commandId.c_str()) &&
                   reaction.arm(command, esp_timer_get_time(), buttonS2.pressed(), buttonS3.pressed())) {
          Serial.printf("[REACTION] armé, délai local=%u ms\n", command.delayMs);
        }
      }
      nextPollMs = nowMs + COMMAND_POLL_INTERVAL_MS;
    }
  }
#endif
  delay(2);  // Yield to the Arduino/Wi-Fi tasks; never used for gameplay timing.
}
