#pragma once

// Set a different 8-63 character setup password for every manufactured box.
// Print it on the box label; never use a shared default in released firmware.
#define PARTYBOX_SETUP_PASSWORD ""
#define PARTYBOX_API_BASE_URL "https://..."
#define PARTYBOX_BOX_ID "PB001"
#define PARTYBOX_DEVICE_TOKEN "..."

// HTTPS is refused when this CA is empty. Copy the PEM root certificate used
// by the public API into secrets.h as a C++ raw string literal.
#define PARTYBOX_TLS_ROOT_CA ""
