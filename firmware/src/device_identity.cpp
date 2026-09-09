#include "device_identity.h"

#include <esp_system.h>

#include "config.h"

void DeviceIdentity::begin() {
  bootId_ = String(static_cast<unsigned long>(esp_random()), HEX) +
            String(static_cast<unsigned long>(ESP.getEfuseMac()), HEX);
}

String DeviceIdentity::nextEventId() {
  ++counter_;
  return String(PARTYBOX_BOX_ID) + "-" + bootId_ + "-" + String(counter_);
}
