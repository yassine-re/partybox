# Firmware PartyBox ESP32-S2

Le firmware cible le module **ESP32-S2-WROVER** du PCB custom Insensia (4 MB de flash, 2 MB de PSRAM annoncés). PlatformIO utilise `esp32-s2-saola-1` comme définition technique compatible ESP32-S2 ; le PCB n’est pas une Saola. Le firmware n’utilise pas la PSRAM et compile avec les quatre GPIO inconnus désactivés.

L’ESP-Prog apparaît sous Linux comme FTDI Dual RS232-H, généralement avec deux interfaces `/dev/ttyUSB*`. L’UART de programmation semble être la seconde interface, mais les numéros changent selon les branchements : aucun port n’est codé dans le dépôt.

## Configuration sûre

Copier le fichier exemple, qui est le seul fichier de secrets versionné :

```sh
cp firmware/include/secrets.example.h firmware/include/secrets.h
```

Renseigner dans `secrets.h` le SSID, le mot de passe Wi-Fi, l’URL publique HTTPS sans slash final, `PB001`, le token device et le certificat racine PEM de l’API. `secrets.h` est ignoré par Git. Le client refuse volontairement HTTPS lorsque `PARTYBOX_TLS_ROOT_CA` est vide ; il ne bascule jamais silencieusement vers une validation TLS non sûre. Aucun log ne contient le mot de passe ou le token.

Le pinout est centralisé dans `include/board_config.h` :

```cpp
BUTTON_S2_PIN = -1;
BUTTON_S3_PIN = -1;
LED_RED_PIN = -1;
LED_GREEN_PIN = -1;
```

`-1` désactive proprement la fonction. Le firmware n’appelle jamais `pinMode`, `digitalRead` ou `digitalWrite` avec une pin invalide. Régler aussi `BUTTON_ACTIVE_LOW` et `LED_ACTIVE_LOW` d’après les mesures réelles. **S2, S3, D19 et Q7 sont des références de composants, pas des numéros GPIO.**

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
7. Après identification électrique, configurer les boutons en entrée et seulement ensuite les LEDs autorisées.
8. Tester enfin un challenge complet. Ne pas activer les sorties LED avant d’avoir une hypothèse physique raisonnable.

## Identifier S2, S3 et les LEDs sans schéma

Ne jamais scanner tous les GPIO en sortie. Avec la carte hors tension :

1. utiliser un multimètre en mode continuité ;
2. identifier les deux bornes commutées de S2 puis S3 ;
3. suivre la piste vers une patte ou un pad du module ESP32-S2-WROVER ;
4. utiliser le pinout officiel du **module** pour convertir ce pad en GPIO ;
5. suivre chaque LED via sa résistance et, le cas échéant, son transistor de commande ;
6. noter si le bouton ou la LED est actif à l’état haut ou bas ;
7. ajouter seulement les hypothèses de boutons dans `DIAGNOSTIC_CANDIDATE_INPUT_PINS` ;
8. ajouter seulement les sorties confirmées dans `DIAGNOSTIC_ALLOWED_LED_PINS`, puis activer temporairement `DIAGNOSTIC_ENABLE_LED_TEST`.

Le diagnostic affiche chaque seconde les entrées explicitement listées. Il ne parcourt jamais les GPIO et ne transforme aucune pin inconnue en sortie.

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

Le backend persiste le prochain déclenchement et crée aléatoirement un solo ou un duel pendant une partie `playing`, uniquement si le heartbeat est récent. L’hôte attribue S2/S3. L’ESP poll toutes les secondes, acquitte `reaction_arm`, allume rouge, attend localement 2 à 6 secondes, allume vert et démarre `esp_timer_get_time()` à cet instant. Aucun appel réseau n’a lieu pendant rouge/vert.

Un appui stable avant vert produit un faux départ. Après vert, le premier front stable gagne ; après cinq secondes, le résultat est un timeout. Le terminal est renvoyé avec un `event_id` stable pendant les retries. Le backend déduplique, calcule les points, met à jour le score sans compléter de mission et notifie les téléphones par WebSocket. Le firmware confirme avec une courte LED verte puis revient en attente.

La reconnexion Wi-Fi utilise un backoff 1–30 secondes. Un reboot fait repoller les commandes `pending` ou `acknowledged` non expirées. Une coupure du boîtier n’interrompt jamais la partie web.

Pour une démonstration plus fréquente, mettre `REACTION_MIN_INTERVAL_SECONDS=15` et `REACTION_MAX_INTERVAL_SECONDS=30`, puis recréer le conteneur backend avec `docker compose up -d --no-deps backend`.

## Dépannage

- `Permission denied` sur ttyUSB : ajouter l’utilisateur au groupe `dialout`, se reconnecter, puis vérifier les permissions ; éviter `sudo pio`.
- `Connecting...` : vérifier IO0/EN, TX/RX, masse commune, alimentation et le bon port ESP-Prog.
- Mauvais port : débrancher/rebrancher et comparer la sortie de `pio device list`.
- `[ERROR] pinout incomplet` : attendu tant que les quatre GPIO ne sont pas identifiés.
- Box hors ligne dans l’UI : vérifier un heartbeat HTTP 200 et `DEVICE_ONLINE_TIMEOUT_SECONDS`.
- HTTP 401 : reprovisionner PB001 et recopier exactement le nouveau token dans `secrets.h`.
- Wi-Fi inaccessible : l’ESP32-S2 utilise le Wi-Fi 2,4 GHz ; vérifier SSID, mot de passe et portée.
- HTTPS refusé : installer le bon certificat racine PEM dans `PARTYBOX_TLS_ROOT_CA` et vérifier l’horloge/certificat du serveur.

La batterie et sa recharge semblent gérées électroniquement par le PCB : aucune lecture ADC ou logique batterie arbitraire n’est implémentée. Le tag NFC reste passif et indépendant ; il doit simplement pointer vers `https://<domain>/box/PB001`. Aucun MQTT, WebSocket ESP, Bluetooth, PIR, NFC actif ou OTA n’est utilisé.
