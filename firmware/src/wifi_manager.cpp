#include "wifi_manager.h"

#include <WiFi.h>
#include <time.h>

#include "config.h"

void PartyBoxWiFi::begin() {
  WiFi.onEvent([](WiFiEvent_t event, WiFiEventInfo_t info) {
    if (event == ARDUINO_EVENT_WIFI_STA_DISCONNECTED) {
      Serial.printf("[WIFI] association interrompue, raison=%d\n",
                    info.wifi_sta_disconnected.reason);
    }
  });
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  Serial.println("[WIFI] gestion non bloquante initialisée");
}

bool PartyBoxWiFi::connected() const { return WiFi.status() == WL_CONNECTED; }

bool PartyBoxWiFi::clockSynchronized() const { return clockSynchronized_; }

void PartyBoxWiFi::loop(unsigned long nowMs) {
  if (connected()) {
    if (!wasConnected_) {
      Serial.printf("[WIFI] connecté, IP=%s RSSI=%d dBm\n", WiFi.localIP().toString().c_str(), WiFi.RSSI());
      wasConnected_ = true;
      if (!clockSyncStarted_) {
        configTime(0, 0, "pool.ntp.org", "time.cloudflare.com");
        clockSyncStarted_ = true;
        Serial.println("[TIME] synchronisation NTP demandée");
      }
    }
    if (!clockSynchronized_ && time(nullptr) >= 1704067200) {
      clockSynchronized_ = true;
      Serial.println("[TIME] horloge synchronisée, HTTPS autorisé");
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
