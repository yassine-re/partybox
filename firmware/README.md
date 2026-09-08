# Firmware PartyBox

Structure PlatformIO / Arduino pour une carte générique ESP32 (`esp32dev`). Adapter la carte au matériel effectivement choisi.

```sh
cd firmware
pio run
pio run --target upload
pio device monitor
```

Ce squelette affiche uniquement un message série. Pas de Wi-Fi, MQTT, NFC actif ni logique de bouton.

Le tag NFC passif contiendra simplement l’URL `https://<domaine>/box/PB001` : il fonctionne indépendamment du firmware.

Contrat réservé : `POST /api/boxes/PB001/events` avec `{"type":"button_press"}`. La route répond volontairement **501** sans modifier la partie. Avant son activation, prévoir une identité matérielle distincte du token joueur, l’antirebond, un identifiant d’événement pour les retries et une règle explicite liant l’appui au joueur concerné.
