#include "wifi_manager.h"

#include <WiFi.h>

#include "config.h"

void PartyBoxWiFi::begin() {
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  Serial.println("[WIFI] gestion non bloquante initialisée");
}

bool PartyBoxWiFi::connected() const { return WiFi.status() == WL_CONNECTED; }

void PartyBoxWiFi::loop(unsigned long nowMs) {
  if (connected()) {
    if (!wasConnected_) {
      Serial.printf("[WIFI] connecté, IP=%s RSSI=%d dBm\n", WiFi.localIP().toString().c_str(), WiFi.RSSI());
      wasConnected_ = true;
    }
    retryMs_ = WIFI_RETRY_MIN_MS;
    return;
  }
  if (wasConnected_) {
    Serial.println("[WIFI] connexion perdue");
    wasConnected_ = false;
  }
  if (static_cast<long>(nowMs - nextAttemptMs_) < 0) return;
  if (String(PARTYBOX_WIFI_SSID) == "...") {
    Serial.println("[ERROR] Wi-Fi non configuré dans include/secrets.h");
  } else {
    Serial.println("[WIFI] tentative de connexion");
    WiFi.begin(PARTYBOX_WIFI_SSID, PARTYBOX_WIFI_PASSWORD);
  }
  nextAttemptMs_ = nowMs + retryMs_;
  retryMs_ = min(retryMs_ * 2, WIFI_RETRY_MAX_MS);
}
