#pragma once

#include <Arduino.h>
#include <DNSServer.h>
#include <Preferences.h>
#include <WebServer.h>

class PartyBoxWiFi {
 public:
  void begin(bool resetCredentials = false);
  void loop(unsigned long nowMs);
  bool connected() const;
  bool clockSynchronized() const;

 private:
  void startSetupPortal();
  void stopSetupPortal();
  void handleSetupSubmission();
  void sendSetupPage(const char* message);
  bool saveCredentials(const String& ssid, const String& password);

  Preferences preferences_;
  DNSServer setupDns_;
  WebServer setupServer_{80};
  String ssid_;
  String password_;
  String pendingSsid_;
  String pendingPassword_;
  String setupNonce_;
  bool setupActive_ = false;
  bool pendingConnection_ = false;
  bool setupFailed_ = false;
  unsigned long pendingDeadlineMs_ = 0;
  unsigned long nextAttemptMs_ = 0;
  unsigned long retryMs_ = 1000;
  bool wasConnected_ = false;
  bool clockSyncStarted_ = false;
  bool clockSynchronized_ = false;
};
