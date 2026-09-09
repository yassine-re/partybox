#pragma once

#define PARTYBOX_WIFI_SSID "..."
#define PARTYBOX_WIFI_PASSWORD "..."
#define PARTYBOX_API_BASE_URL "https://..."
#define PARTYBOX_BOX_ID "PB001"
#define PARTYBOX_DEVICE_TOKEN "..."

// HTTPS is refused when this CA is empty. Copy the PEM root certificate used
// by the public API into secrets.h as a C++ raw string literal.
#define PARTYBOX_TLS_ROOT_CA ""
