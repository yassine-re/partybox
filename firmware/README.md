# Firmware PartyBox ESP32-S2

Le firmware cible le module **ESP32-S2-WROVER** du PCB custom Insensia vB.1 (4 MB de flash détectés). PlatformIO utilise `esp32-s2-saola-1` comme définition technique compatible ESP32-S2 ; le PCB n’est pas une Saola. Le firmware n’utilise pas la PSRAM.

L’ESP-Prog apparaît sous Linux comme FTDI Dual RS232-H, généralement avec deux interfaces `/dev/ttyUSB*`. L’UART de programmation semble être la seconde interface, mais les numéros changent selon les branchements : aucun port n’est codé dans le dépôt.

## Configuration sûre

Copier le fichier exemple, qui est le seul fichier de secrets versionné :

```sh
cp firmware/include/secrets.example.h firmware/include/secrets.h
```

Renseigner dans `secrets.h` le SSID, le mot de passe Wi-Fi, l’URL publique HTTPS sans slash final, `PB001`, le token device et le certificat racine PEM de l’API. `secrets.h` est ignoré par Git. Le client refuse volontairement HTTPS lorsque `PARTYBOX_TLS_ROOT_CA` est vide ; il ne bascule jamais silencieusement vers une validation TLS non sûre. Aucun log ne contient le mot de passe ou le token.

Le pinout mesuré est centralisé dans `include/board_config.h` :

```cpp
BUTTON_S2_PIN = 16;
BUTTON_S3_PIN = 15;
I2C_SCL_PIN = 2;
I2C_SDA_PIN = 3;
RGB_LED_ENABLE_PIN = 33;
RGB_LED_I2C_ADDRESS = 0x38;
```

S2 et S3 sont actifs à l’état bas. D19 est pilotée par le NCP5623 : ses canaux 1, 2 et 3 correspondent respectivement à rouge, vert et bleu. Q10 commute son anode depuis VBB et s’active par `GPIO33` à l’état bas. Le bus I²C est partagé avec un second périphérique détecté à l’adresse `0x43`, auquel le pilote RGB n’écrit jamais. **S2, S3, D19 et Q10 sont des références de composants, pas des numéros GPIO.**

## Build, tests et ports

Depuis la racine du dépôt :

```sh
pio device list
pio test -d firmware -e native
pio run -d firmware -e esp32s2
pio run -d firmware -e esp32s2 -t upload --upload-port /dev/ttyUSB1
pio device monitor --port /dev/ttyUSB1 -b 115200
```

Remplacer `/dev/ttyUSB1` par l’interface détectée. Le build `native` teste la machine d’état et le debounce sur l’ordinateur. La CI compile mais ne flashe jamais de carte.

## Premier flash avec ESP-Prog

Procéder par étapes, alimentation coupée pendant toute modification du câblage :

1. Vérifier tension, masse commune, TX/RX croisés et lignes de boot/reset selon le câblage ESP-Prog déjà présent. L’ESP-Prog ne doit pas alimenter en conflit une carte déjà alimentée.
2. Lancer `pio device list`, puis compiler le build série sans réseau : `pio run -d firmware -e esp32s2-serial`.
3. Mettre la carte en bootloader avec les commandes EN/IO0 du programmateur si nécessaire, puis flasher : `pio run -d firmware -e esp32s2-serial -t upload --upload-port <PORT>`.
4. Ouvrir le moniteur à 115200 bauds. Vérifier les logs `[BOOT]`, le modèle, le reset reason, les 4 MB de flash et la PSRAM détectée. À ce stade aucun GPIO inconnu n’est piloté.
5. Renseigner uniquement le Wi-Fi dans `secrets.h`, compiler `esp32s2` et observer `[WIFI]`. Une URL API volontairement non renseignée permet de tester le Wi-Fi seul.
6. Provisionner le device côté backend, renseigner l’URL, le token et le certificat, reflasher puis vérifier `[HEARTBEAT] HTTP 200`.
7. Vérifier dans les logs S2=`GPIO16`, S3=`GPIO15`, le NCP5623 à `0x38` et `[RGB] NCP5623 prêt`.
8. Tester enfin un challenge complet.

## Pinout confirmé sans schéma

Les mesures ont été effectuées sans jamais scanner les GPIO en sortie :

1. utiliser un multimètre en mode continuité ;
2. identifier les deux bornes commutées de S2 puis S3 ;
3. suivre la piste vers une patte ou un pad du module ESP32-S2-WROVER ;
4. utiliser le pinout officiel du **module** pour convertir ce pad en GPIO ;
5. suivre chaque LED via sa résistance et, le cas échéant, son transistor de commande ;
6. noter si le bouton ou la LED est actif à l’état haut ou bas ;
7. confirmer S2 sur `GPIO16` et S3 sur `GPIO15` avec le diagnostic d’entrée ;
8. scanner le bus I²C, puis tester le NCP5623 à faible courant après avoir tracé Q10 sur `GPIO33`.

Le diagnostic affiche chaque seconde les entrées explicitement listées et recense les adresses I²C. Il ne parcourt jamais les GPIO en sortie. Le pilote de D19 initialise les trois PWM à zéro avant d’activer Q10, puis n’écrit que sur l’adresse `0x38` lors d’un changement d’état.

## Provisionner PB001

Après application de la migration `008`, générer un token aléatoire depuis l’image backend :

```sh
docker compose build backend
docker compose run --rm --entrypoint /app/provision-device backend -box-id PB001
```

La commande affiche le token brut une seule fois. La base ne conserve que son hash SHA-256. Pour remplacer le token par une valeur déjà générée, fournir `DEVICE_TOKEN` dans l’environnement de la commande ; ne pas le mettre dans `.env.example`, l’historique shell ou un log.

Test heartbeat manuel :

```sh
curl -i -X POST 'https://<domain>/api/device/boxes/PB001/heartbeat' \
  -H 'Authorization: Bearer <DEVICE_TOKEN>' \
  -H 'Content-Type: application/json' \
  -d '{"firmware_version":"manual-test","uptime_ms":1000,"wifi_rssi":-50}'
```

Polling manuel :

```sh
curl -i 'https://<domain>/api/device/boxes/PB001/commands' \
  -H 'Authorization: Bearer <DEVICE_TOKEN>'
```

## Fonctionnement du challenge

Le backend persiste le prochain déclenchement et crée aléatoirement un solo ou un duel pendant une partie `playing`, uniquement si le heartbeat est récent. L’hôte attribue les boutons bleu (S2) et rouge (S3). L’ESP poll toutes les secondes, acquitte `reaction_arm`, allume rouge, attend localement 2 à 6 secondes, allume vert et démarre `esp_timer_get_time()` à cet instant. Aucun appel réseau n’a lieu pendant rouge/vert.

Un appui stable avant vert produit un faux départ. Après vert, le premier front stable gagne ; après cinq secondes, le résultat est un timeout. Le terminal est renvoyé avec un `event_id` stable pendant les retries. Le backend déduplique, calcule les points, met à jour le score sans compléter de mission et notifie les téléphones par WebSocket. Le firmware confirme avec une courte LED verte puis revient en attente.

La reconnexion Wi-Fi utilise un backoff 1–30 secondes. Après la connexion, le firmware synchronise son horloge par NTP avant d'autoriser HTTPS, afin de vérifier correctement les dates de la chaîne TLS. Un reboot fait repoller les commandes `pending` ou `acknowledged` non expirées. Une coupure du boîtier n’interrompt jamais la partie web.

Pour une démonstration plus fréquente, mettre `REACTION_MIN_INTERVAL_SECONDS=15` et `REACTION_MAX_INTERVAL_SECONDS=30`, puis recréer le conteneur backend avec `docker compose up -d --no-deps backend`.

## Dépannage

- `Permission denied` sur ttyUSB : ajouter l’utilisateur au groupe `dialout`, se reconnecter, puis vérifier les permissions ; éviter `sudo pio`.
- `Connecting...` : vérifier IO0/EN, TX/RX, masse commune, alimentation et le bon port ESP-Prog.
- Mauvais port : débrancher/rebrancher et comparer la sortie de `pio device list`.
- `[RGB] initialisation NCP5623 impossible` : vérifier VBB, SDA=`GPIO3`, SCL=`GPIO2`, l’adresse `0x38` et l’activation Q10=`GPIO33`.
- Box hors ligne dans l’UI : vérifier un heartbeat HTTP 200 et `DEVICE_ONLINE_TIMEOUT_SECONDS`.
- HTTP 401 : reprovisionner PB001 et recopier exactement le nouveau token dans `secrets.h`.
- Wi-Fi inaccessible : l’ESP32-S2 utilise le Wi-Fi 2,4 GHz ; vérifier SSID, mot de passe et portée.
- `[TIME]` ne confirme jamais la synchronisation : vérifier que le réseau autorise DNS et NTP sortants ; aucun appel HTTPS n'est tenté avec une horloge invalide.
- HTTPS refusé : installer le bon certificat racine PEM dans `PARTYBOX_TLS_ROOT_CA` et vérifier l’horloge/certificat du serveur.

La batterie et sa recharge semblent gérées électroniquement par le PCB : aucune lecture ADC ou logique batterie arbitraire n’est implémentée. Le tag NFC reste passif et indépendant ; il doit simplement pointer vers `https://<domain>/box/PB001`. Aucun MQTT, WebSocket ESP, Bluetooth, PIR, NFC actif ou OTA n’est utilisé.
