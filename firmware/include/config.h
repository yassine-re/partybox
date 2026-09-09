#pragma once

#if __has_include("secrets.h")
#include "secrets.h"
#define PARTYBOX_HAS_LOCAL_SECRETS 1
#else
#include "secrets.example.h"
#define PARTYBOX_HAS_LOCAL_SECRETS 0
#endif

#ifndef PARTYBOX_FIRMWARE_VERSION
#define PARTYBOX_FIRMWARE_VERSION "dev"
#endif

#ifndef PARTYBOX_NETWORK_ENABLED
#define PARTYBOX_NETWORK_ENABLED 1
#endif

constexpr unsigned long WIFI_RETRY_MIN_MS = 1000;
constexpr unsigned long WIFI_RETRY_MAX_MS = 30000;
constexpr unsigned long HEARTBEAT_INTERVAL_MS = 5000;
constexpr unsigned long COMMAND_POLL_INTERVAL_MS = 1000;
constexpr unsigned long HTTP_TIMEOUT_MS = 5000;
