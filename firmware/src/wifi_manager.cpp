#include "wifi_manager.h"

#include <WiFi.h>
#include <esp_system.h>
#include <time.h>

#include "config.h"

namespace {
bool validSsid(const String& ssid) {
  if (ssid.isEmpty() || ssid.length() > 32) return false;
  for (size_t i = 0; i < ssid.length(); ++i) {
    if (static_cast<unsigned char>(ssid[i]) < 32) return false;
  }
  return true;
}

bool validPassword(const String& password) {
  if (password.isEmpty()) return true;  // Open networks are supported.
  if (password.length() < 8 || password.length() > 63) return false;
  for (size_t i = 0; i < password.length(); ++i) {
    if (password[i] < 32 || password[i] > 126) return false;
  }
  return true;
}
}  // namespace

void PartyBoxWiFi::begin(bool resetCredentials) {
  WiFi.onEvent([](WiFiEvent_t event, WiFiEventInfo_t info) {
    if (event == ARDUINO_EVENT_WIFI_STA_DISCONNECTED) {
      Serial.printf("[WIFI] association interrompue, raison=%d\n",
                    info.wifi_sta_disconnected.reason);
    }
  });
  WiFi.persistent(false);
  if (!preferences_.begin("pb-wifi", false)) {
    Serial.println("[ERROR] stockage Wi-Fi indisponible");
  }
  if (resetCredentials) {
    preferences_.remove("ssid");
    preferences_.remove("password");
    WiFi.disconnect(true, true);
    Serial.println("[WIFI] configuration Wi-Fi effacée par les deux boutons");
  }
  ssid_ = preferences_.getString("ssid", "");
  password_ = preferences_.getString("password", "");
  if (!validSsid(ssid_) || !validPassword(password_)) {
    ssid_ = "";
    password_ = "";
    startSetupPortal();
    return;
  }
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  Serial.println("[WIFI] configuration locale chargée");
}

void PartyBoxWiFi::startSetupPortal() {
  const String setupPassword(PARTYBOX_SETUP_PASSWORD);
  if (setupPassword.length() < 8 || setupPassword.length() > 63 ||
      !validPassword(setupPassword)) {
    Serial.println("[ERROR] mot de passe de configuration unique requis dans secrets.h");
    return;
  }
  WiFi.mode(WIFI_AP_STA);
  WiFi.setAutoReconnect(false);
  String apSsid = String("PartyBox-") + WiFi.macAddress();
  apSsid.replace(":", "");
  if (!WiFi.softAP(apSsid.c_str(), setupPassword.c_str())) {
    Serial.println("[ERROR] impossible de démarrer le point d'accès de configuration");
    return;
  }
  if (!setupDns_.start(53, "*", WiFi.softAPIP())) {
    Serial.println("[WIFI] portail DNS indisponible ; ouvrir l'adresse IP manuellement");
  }
  setupNonce_ = String(esp_random(), HEX) + String(esp_random(), HEX);
  setupServer_.on("/", HTTP_GET, [this]() { sendSetupPage(""); });
  setupServer_.on("/", HTTP_POST, [this]() { handleSetupSubmission(); });
  setupServer_.on("/status", HTTP_GET, [this]() {
    sendSetupPage(pendingConnection_ ? "Connexion en cours. Si le réseau PartyBox disparaît, la configuration a réussi."
                                      : setupFailed_ ? "Connexion impossible. Vérifiez le nom du réseau et le mot de passe."
                                                     : "Saisissez le Wi-Fi à utiliser.");
  });
  setupServer_.onNotFound([this]() {
    setupServer_.sendHeader("Location", "http://192.168.4.1/", true);
    setupServer_.send(302, "text/plain", "");
  });
  setupServer_.begin();
  setupActive_ = true;
  Serial.printf("[WIFI] configuration sur %s : http://192.168.4.1\n", apSsid.c_str());
}

void PartyBoxWiFi::sendSetupPage(const char* message) {
  String page = F(R"html(<!doctype html><html lang="fr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="theme-color" content="#111410"><title>Configurer PartyBox</title><style>
:root{color-scheme:dark;--bg:#111410;--panel:#1b1f18;--text:#f1f3e8;--muted:#a3aa96;--line:#333a2b;--lime:#c7ff5e;--pink:#f9a8cb}
*{box-sizing:border-box}body{margin:0;min-height:100vh;background:var(--bg);color:var(--text);font:16px/1.55 "Avenir Next","Trebuchet MS",sans-serif}
main{width:min(100% - 32px,480px);margin:0 auto;padding:48px 0 64px}.brand{color:var(--lime);font-size:13px;font-weight:800;letter-spacing:.16em;text-transform:uppercase}
.card{margin-top:25px;padding:clamp(24px,6vw,36px);border:1px solid var(--line);border-radius:18px;background:var(--panel);box-shadow:0 24px 80px #0008}
h1{margin:0 0 12px;font-size:clamp(32px,8vw,46px);line-height:1.06;letter-spacing:-.055em}p{margin:0 0 24px;color:var(--muted)}.message{color:var(--pink)}
label{display:block;margin:20px 0 0;font-size:14px;font-weight:700}input{display:block;width:100%;min-height:54px;margin-top:8px;padding:14px 16px;border:1px solid #43463a;border-radius:7px;background:#181b15;color:var(--text);font:inherit}
input:focus-visible,button:focus-visible{outline:3px solid var(--pink);outline-offset:3px}.hint{margin:9px 0 0;font-size:12px}button{width:100%;min-height:54px;margin-top:28px;padding:12px 18px;border:0;border-radius:8px;background:var(--lime);color:var(--bg);font:inherit;font-weight:800;cursor:pointer}.foot{margin:20px 0 0;text-align:center;font-size:12px}
</style></head><body><main><div class="brand">PartyBox <span aria-hidden="true">✦</span> Configuration</div><section class="card"><h1>Connectons la box.</h1><p>Choisissez le Wi-Fi auquel PartyBox doit se connecter.</p><p class="message" role="status">)html");
  page += message;
  page += F(R"html(</p><form method="post" action="/"><label for="ssid">Nom du réseau Wi-Fi</label><input id="ssid" name="ssid" required maxlength="32" autocomplete="off"><label for="password">Mot de passe Wi-Fi</label><input id="password" name="password" type="password" maxlength="63" autocomplete="off"><p class="hint">Laissez vide uniquement pour un réseau ouvert.</p><input type="hidden" name="nonce" value=")html");
  page += setupNonce_;
  page += F(R"html("><button type="submit">Connecter la box</button></form></section><p class="foot">Wi-Fi 2,4 GHz uniquement · Vos identifiants restent dans la box.</p></main></body></html>)html");
  setupServer_.sendHeader("Cache-Control", "no-store");
  setupServer_.send(200, "text/html; charset=utf-8", page);
}

void PartyBoxWiFi::handleSetupSubmission() {
  if (!setupServer_.hasArg("nonce") || setupServer_.arg("nonce") != setupNonce_ ||
      !setupServer_.hasArg("ssid") || !setupServer_.hasArg("password")) {
    setupServer_.send(400, "text/plain", "Formulaire invalide");
    return;
  }
  if (pendingConnection_) {
    setupServer_.send(409, "text/plain", "Connexion deja en cours");
    return;
  }
  const String ssid = setupServer_.arg("ssid");
  const String password = setupServer_.arg("password");
  if (!validSsid(ssid) || !validPassword(password)) {
    setupServer_.send(400, "text/plain", "SSID (1-32 octets) ou mot de passe (vide ou 8-63 caracteres ASCII) invalide");
    return;
  }
  pendingSsid_ = ssid;
  pendingPassword_ = password;
  pendingConnection_ = true;
  setupFailed_ = false;
  pendingDeadlineMs_ = millis() + 20000;
  WiFi.begin(pendingSsid_.c_str(), pendingPassword_.c_str());
  sendSetupPage("Connexion en cours. Si le réseau PartyBox disparaît, la configuration a réussi. Sinon, rechargez cette page après 20 secondes.");
}

bool PartyBoxWiFi::saveCredentials(const String& ssid, const String& password) {
  preferences_.putString("password", password);
  preferences_.putString("ssid", ssid);
  return preferences_.getString("ssid", "") == ssid &&
         preferences_.getString("password", "") == password;
}

void PartyBoxWiFi::stopSetupPortal() {
  setupDns_.stop();
  setupServer_.stop();
  WiFi.softAPdisconnect(true);
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  setupActive_ = false;
  pendingConnection_ = false;
  pendingSsid_ = "";
  pendingPassword_ = "";
  setupNonce_ = "";
  Serial.println("[WIFI] configuration enregistrée, point d'accès fermé");
}

bool PartyBoxWiFi::connected() const { return !setupActive_ && WiFi.status() == WL_CONNECTED; }

bool PartyBoxWiFi::clockSynchronized() const { return clockSynchronized_; }

void PartyBoxWiFi::loop(unsigned long nowMs) {
  if (setupActive_) {
    setupDns_.processNextRequest();
    setupServer_.handleClient();
    if (pendingConnection_) {
      if (WiFi.status() == WL_CONNECTED && WiFi.SSID() == pendingSsid_) {
        if (saveCredentials(pendingSsid_, pendingPassword_)) {
          ssid_ = pendingSsid_;
          password_ = pendingPassword_;
          stopSetupPortal();
        } else {
          Serial.println("[ERROR] enregistrement Wi-Fi impossible");
          WiFi.disconnect(false, false);
          pendingConnection_ = false;
          setupFailed_ = true;
        }
      } else if (static_cast<long>(nowMs - pendingDeadlineMs_) >= 0) {
        WiFi.disconnect(false, false);
        pendingConnection_ = false;
        setupFailed_ = true;
        Serial.println("[WIFI] configuration : connexion impossible");
      }
    }
    if (setupActive_) return;
  }
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
  if (ssid_.isEmpty() || static_cast<long>(nowMs - nextAttemptMs_) < 0) return;
  Serial.println("[WIFI] tentative de connexion");
  WiFi.begin(ssid_.c_str(), password_.c_str());
  nextAttemptMs_ = nowMs + retryMs_;
  retryMs_ = min(retryMs_ * 2, WIFI_RETRY_MAX_MS);
}
