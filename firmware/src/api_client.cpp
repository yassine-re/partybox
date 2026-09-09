#include "api_client.h"

#include <ArduinoJson.h>
#include <HTTPClient.h>
#include <WiFiClientSecure.h>

#include "config.h"

namespace {
bool successful(int status) { return status >= 200 && status < 300; }

String devicePath(const String& suffix) {
  return "/api/device/boxes/" + String(PARTYBOX_BOX_ID) + suffix;
}
}  // namespace

int ApiClient::request(const char* method, const String& path, const String& body, String& response) {
  const String base(PARTYBOX_API_BASE_URL);
  if (base == "https://..." || String(PARTYBOX_DEVICE_TOKEN) == "...") {
    Serial.println("[ERROR] API ou token device non configuré");
    return -1;
  }
  const String url = base + path;
  HTTPClient http;
  WiFiClientSecure secure;
  bool opened = false;
  if (url.startsWith("https://")) {
    if (strlen(PARTYBOX_TLS_ROOT_CA) == 0) {
      Serial.println("[ERROR] HTTPS refusé : PARTYBOX_TLS_ROOT_CA est vide");
      return -1;
    }
    secure.setCACert(PARTYBOX_TLS_ROOT_CA);
    opened = http.begin(secure, url);
  } else {
    opened = http.begin(url);
  }
  if (!opened) return -1;
  http.setConnectTimeout(HTTP_TIMEOUT_MS);
  http.setTimeout(HTTP_TIMEOUT_MS);
  http.addHeader("Authorization", "Bearer " + String(PARTYBOX_DEVICE_TOKEN));
  int status;
  if (strcmp(method, "GET") == 0) {
    status = http.GET();
  } else {
    http.addHeader("Content-Type", "application/json");
    status = http.POST(body);
  }
  if (status > 0) response = http.getString();
  http.end();
  return status;
}

bool ApiClient::heartbeat(uint64_t uptimeMs, int wifiRssi) {
  JsonDocument document;
  document["firmware_version"] = PARTYBOX_FIRMWARE_VERSION;
  document["uptime_ms"] = uptimeMs;
  document["wifi_rssi"] = wifiRssi;
  String body;
  serializeJson(document, body);
  String response;
  const int status = request("POST", devicePath("/heartbeat"), body, response);
  Serial.printf("[HEARTBEAT] HTTP %d\n", status);
  return successful(status);
}

bool ApiClient::pollCommand(ReactionCommand& command) {
  String response;
  const int status = request("GET", devicePath("/commands"), "", response);
  if (!successful(status)) {
    Serial.printf("[API] polling HTTP %d\n", status);
    return false;
  }
  JsonDocument document;
  if (deserializeJson(document, response)) {
    Serial.println("[ERROR] réponse commands JSON invalide");
    return false;
  }
  JsonArray commands = document["commands"].as<JsonArray>();
  if (commands.size() == 0) return false;
  JsonObject value = commands[0];
  if (String(value["type"].as<const char*>()) != "reaction_arm") return false;
  JsonObject payload = value["payload"];
  command.commandId = value["id"].as<const char*>();
  command.challengeId = value["challenge_id"].as<const char*>();
  command.duel = String(payload["kind"].as<const char*>()) == "duel";
  command.s2Enabled = payload["button_s2_enabled"] | false;
  command.s3Enabled = payload["button_s3_enabled"] | false;
  command.delayMs = payload["delay_ms"] | 0;
  return true;
}

bool ApiClient::acknowledge(const String& commandId) {
  String response;
  const int status = request("POST", devicePath("/commands/" + commandId + "/ack"), "{}", response);
  Serial.printf("[COMMAND] ack HTTP %d\n", status);
  return successful(status);
}

bool ApiClient::submitReaction(const ReactionOutcome& outcome, const String& eventId) {
  JsonDocument document;
  document["event_id"] = eventId;
  document["challenge_id"] = outcome.challengeId.c_str();
  document["command_id"] = outcome.commandId.c_str();
  document["false_start"] = outcome.falseStart;
  document["timeout"] = outcome.timeout;
  if (!outcome.button.empty()) document["button"] = outcome.button.c_str();
  if (outcome.reactionMs >= 0) document["reaction_ms"] = outcome.reactionMs;
  String body;
  serializeJson(document, body);
  String response;
  const int status = request("POST", devicePath("/reaction-results"), body, response);
  Serial.printf("[REACTION] résultat HTTP %d\n", status);
  return successful(status);
}
