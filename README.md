# PartyBox

PartyBox transforme une soirée IRL en jeu de missions secrètes, de chasse au trésor ou en partie Chaos aux règles dynamiques. Un tag NFC passif dans le boîtier ouvre une URL comme `https://partybox.example.com/box/PB001`. Chaque joueur utilise son téléphone ; Go et PostgreSQL centralisent les parties, missions, scores et événements métier sur un serveur.

Ce dépôt contient le premier MVP logiciel : création de partie, lobby partagé, démarrage par l’hôte, missions privées, validation virtuelle, points, nouvelle mission et classement final. Le pipeline ML entraîne hors ligne un ranking contextuel optionnel ; le firmware conserve uniquement son emplacement préparé.

## Démarrage rapide

Prérequis : Git, Docker Engine avec Docker Compose v2+ (ou Docker Desktop démarré). Aucun Go, Node ou PostgreSQL local n’est nécessaire avec Compose.

Depuis la racine du dépôt :

```sh
cp .env.example .env
docker compose up --build
```

Ouvrir **[http://localhost:3000/box/PB001](http://localhost:3000/box/PB001)**.

Compose attend PostgreSQL, applique les migrations, exécute le seed idempotent, puis démarre l’API et le frontend. La box `PB001`, **18 missions secrètes**, **15 défis de chasse au trésor** et **15 missions Chaos** sont créés automatiquement. Aucune partie ni joueur de démonstration n’est créé. La génération IA reste optionnelle et désactivée tant que `OPENAI_API_KEY` et `OPENAI_MODEL` ne sont pas renseignés.

Pour lancer en arrière-plan, suivre les logs ou arrêter :

```sh
docker compose up --build -d
docker compose logs -f backend frontend
docker compose down
```

`down` conserve les données dans le volume PostgreSQL. L’option `-v` supprimerait ces données. Après modification du code, relancer `docker compose up --build -d` : les conteneurs utilisent les builds du projet, sans hot reload.

## Architecture

```text
Téléphones / navigateur
       │ URL NFC : /box/PB001
       ▼
Caddy HTTPS (profil production, optionnel en local)
       │
       ▼
SvelteKit + TypeScript :3000
       │ REST /api + upgrade WebSocket de même origine
       ▼
Go + Gin :8080
       │ pgx / transactions SQL
       ▼
PostgreSQL :5432
```

- **Frontend** : SvelteKit 2, Svelte 5, TypeScript, CSS simple et composants réutilisables. Interface mobile sombre, accents citron et rose. Aucun framework CSS ni gestionnaire d’état supplémentaire. Le build utilise `adapter-node` pour le déploiement sur VPS.
- **Backend** : les handlers gèrent HTTP/JSON, les services gèrent les règles et l’enchaînement transactionnel, les repositories contiennent le SQL et les verrous PostgreSQL. Pas d’ORM.
- **IA** : `internal/ai` implémente un fournisseur de contenu derrière `MissionGenerator`. OpenAI produit un catalogue JSON structuré ; PartyBox conserve l’attribution, la validation, les scores et le fallback vers le seed.
- **Communication** : REST reste la source de vérité via `frontend/src/lib/api/`. Un WebSocket authentifié par ticket signale uniquement qu’une ressource a changé, puis le frontend la relit via REST. Le bouton de rafraîchissement et la resynchronisation au retour dans l’onglet sont conservés ; un fallback à 25 secondes ne tourne que lorsque le socket est indisponible.
- **Réseau** : le navigateur appelle `/api` sur son propre domaine. La passerelle SvelteKit relaie vers `backend` sur le réseau privé Compose. Le frontend est accessible sur le réseau de développement ; PostgreSQL publie uniquement un port local sur `127.0.0.1` pour les explorateurs de base de données. L’API Go ne publie pas de port sur l’hôte.
- **Persistance** : migrations SQL versionnées, seed idempotent et volume PostgreSQL. Le feedback mission reste lié à son assignment et alimente un export ML hors ligne. Les conteneurs API et frontend tournent avec des utilisateurs sans privilèges.

Le dossier courant est directement la racine PartyBox, même s’il porte un autre nom localement ; aucun sous-dossier `partybox/` supplémentaire n’est nécessaire.

## Arborescence

```text
.
├── backend/
│   ├── cmd/api/main.go          # Serveur et commandes migrate / seed
│   ├── internal/
│   │   ├── ai/                  # Provider Responses, prompts et validation du catalogue
│   │   ├── database/            # Pool, exécution des migrations et du seed
│   │   ├── handlers/            # Composition HTTP + routes boxes/games/players/realtime
│   │   ├── chaos/               # Moteur, événements et état du mode Chaos
│   │   ├── realtime/            # Tickets courts, hub par partie et clients WebSocket
│   │   ├── ml/                  # Chargement et scoring du modèle portable optionnel
│   │   ├── services/            # Identités et cycle de vie d’une partie
│   │   ├── repositories/        # SQL, transactions et journal d’événements
│   │   └── models/              # Types métier, réponses JSON et GameEvent persistant
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/lib/api/             # Client fetch, types et sessions locales
│   ├── src/lib/realtime/        # Connexion, backoff et notifications entrantes
│   ├── src/lib/components/      # Orchestration de l’expérience et composants partagés
│   │   └── game/                # En-tête, lobby, jeu, bannière Chaos et résultats
│   ├── src/routes/
│   │   ├── +page.svelte         # Accueil et saisie d’un code box
│   │   ├── box/[boxId]/         # Entrée → lobby → jeu → fin
│   │   └── api/[...path]/       # Passerelle vers Go
│   ├── src/app.css
│   ├── src/theme.css
│   ├── static/favicon.svg
│   ├── Dockerfile
│   └── package-lock.json
├── database/
│   ├── migrations/001_initial.{up,down}.sql
│   ├── migrations/002_game_modes.{up,down}.sql
│   ├── migrations/003_game_events.{up,down}.sql
│   ├── migrations/004_ai_missions.{up,down}.sql
│   ├── migrations/005_chaos_mode.{up,down}.sql
│   ├── migrations/006_treasure_photo_validation.{up,down}.sql
│   ├── migrations/007_mission_feedback.{up,down}.sql
│   └── seed.sql
├── firmware/
│   ├── platformio.ini
│   ├── src/main.cpp
│   └── include/
├── ml/
│   ├── dataset.py              # Jointures et contrôles qualité
│   ├── export_dataset.py       # Export CSV et statistiques
│   ├── features.py             # Features pré-assignment sans fuite
│   ├── train.py                # Split groupé, Logistic Regression et artifacts
│   ├── evaluate.py             # Baseline, métriques et coefficients
│   ├── predict.py              # Scoring et ranking Python
│   └── tests/                  # Tests et fixtures Python ↔ Go
├── infra/Caddyfile
├── docker-compose.yml
├── .env.example
├── .gitignore
└── README.md
```

Le lobby, le jeu, le classement et la fin partagent `/box/[boxId]` : le même lien NFC reste pertinent à chaque étape. Les composants sont séparés sans multiplier les routes ni les identifiants dans les liens partagés.

## Configuration

`.env` est ignoré par Git. Les valeurs livrées sont uniquement destinées au développement. Ne jamais mettre un secret dans une variable `PUBLIC_*`, car elle est accessible au navigateur.

| Variable            | Valeur de développement                                                        | Rôle                                                                            |
| ------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------- |
| `DATABASE_URL`      | `postgres://partybox:partybox_dev_only@postgres:5432/partybox?sslmode=disable` | Connexion pgx depuis Docker                                                     |
| `POSTGRES_DB`       | `partybox`                                                                     | Base créée au premier démarrage PostgreSQL                                      |
| `POSTGRES_USER`     | `partybox`                                                                     | Utilisateur PostgreSQL                                                          |
| `POSTGRES_PORT`     | `5432`                                                                         | Port PostgreSQL publié uniquement sur `127.0.0.1`                               |
| `POSTGRES_PASSWORD` | `partybox_dev_only`                                                            | Mot de passe de développement                                                   |
| `BACKEND_PORT`      | `8080`                                                                         | Port interne Go                                                                 |
| `FRONTEND_PORT`     | `3000`                                                                         | Port du frontend publié sur l’hôte                                              |
| `FRONTEND_BIND`     | `0.0.0.0`                                                                      | Adresse d’écoute sur l’hôte ; utiliser `127.0.0.1` derrière Caddy en production |
| `FRONTEND_URL`      | `http://localhost:3000`                                                        | Origine publique pour SvelteKit et CORS de Go                                   |
| `PUBLIC_API_URL`    | `/api`                                                                         | Base d’URL pour le navigateur, configurée au démarrage                          |
| `API_INTERNAL_URL`  | `http://backend:8080`                                                          | Backend joint par le serveur SvelteKit                                          |
| `OPENAI_API_KEY`    | vide                                                                           | Clé backend optionnelle, jamais exposée au navigateur                           |
| `OPENAI_MODEL`      | vide                                                                           | Modèle compatible Responses API et Structured Outputs ; obligatoire avec la clé |
| `DOMAIN`            | `partybox.example.com`                                                         | Domaine du profil Caddy                                                         |
| `ACME_EMAIL`        | `you@example.com`                                                              | Contact pour les certificats HTTPS                                              |

Maintenir `DATABASE_URL` cohérent avec les trois variables PostgreSQL. Si le mot de passe contient des caractères réservés aux URL, les encoder dans `DATABASE_URL`. Maintenir également `API_INTERNAL_URL` cohérent avec `BACKEND_PORT` et `FRONTEND_URL` avec l’URL utilisée par les joueurs.

Changer `POSTGRES_PASSWORD` dans `.env` ne modifie pas le mot de passe d’une base déjà initialisée : effectuer sa rotation dans PostgreSQL aussi.

Le binaire Go accepte également `MIGRATIONS_DIR` et `SEED_FILE`, configurées dans son image Docker. SvelteKit utilise `HOST=0.0.0.0`, `PORT=3000` et `BODY_SIZE_LIMIT=6M` dans le conteneur pour les photos. La passerelle garde une limite de 8 Ko pour les autres routes. En lancement natif, il faut charger les variables dans le shell ; le backend ne lit pas automatiquement `.env`.

## Connexion avec un explorateur de base de données

PostgreSQL est accessible depuis un explorateur installé sur le même ordinateur (DBeaver, DataGrip, TablePlus ou extension d’éditeur) avec ces paramètres :

| Paramètre         | Valeur                                       |
| ----------------- | -------------------------------------------- |
| Type de connexion | PostgreSQL                                   |
| Hôte              | `127.0.0.1`                                  |
| Port              | `5432` (ou `POSTGRES_PORT` dans `.env`)      |
| Base              | `partybox` (ou `POSTGRES_DB`)                |
| Utilisateur       | `partybox` (ou `POSTGRES_USER`)              |
| Mot de passe      | La valeur de `POSTGRES_PASSWORD` dans `.env` |
| SSL               | Désactivé pour cette connexion locale        |
| Schéma            | `public`                                     |

Le nom d’hôte `postgres` de `DATABASE_URL` est réservé au réseau Docker. Depuis un explorateur local, utiliser `127.0.0.1`, sans URL zrok ni préfixe HTTP. La publication du port est limitée à la boucle locale : elle n’expose pas la base sur le Wi-Fi ou Internet.

Si le port 5432 est déjà occupé, définir par exemple `POSTGRES_PORT=5433`, puis appliquer avec `docker compose up -d postgres`. Utiliser alors 5433 dans l’explorateur ; `DATABASE_URL` du backend conserve `postgres:5432`, le port interne Docker.

## Migrations et seed

Les deux opérations sont automatiques avec le démarrage Compose. Pour les relancer explicitement sur une stack déjà démarrée :

```sh
docker compose run --rm migrate
docker compose run --rm seed
```

`migrate` applique les fichiers `*.up.sql` par ordre lexical et inscrit leurs versions dans `schema_migrations`. Les migrations et le suivi des versions partagent une transaction ; un verrou évite deux exécutions simultanées. Ajouter une nouvelle migration numérotée plutôt que modifier une migration déjà appliquée.

Le seed insère `PB001`, 18 missions secrètes, 15 défis de chasse au trésor et 15 missions Chaos avec points, catégorie et difficulté (1 à 3), sans dupliquer les lignes ni remplacer les données existantes. Les missions déjà jouées sont évitées tant qu’il reste des missions inédites dans le mode choisi ; après épuisement du catalogue, la moins récemment attribuée revient.

Les migrations `004_ai_missions` et `005_chaos_mode` restent séparées afin que les catalogues IA et l’état Chaos puissent évoluer indépendamment. `007_mission_feedback` crée la table de notation ; le numéro `006` est volontairement réservé à la branche de validation photo Treasure Hunt.

Le fichier `.down.sql` est fourni pour un retour arrière manuel. Le binaire n’exécute pas de rollback automatique. Le rollback de `004_ai_missions` supprime les catalogues IA et leurs attributions, car l’ancien schéma ne peut pas représenter des missions propres à une partie. Un rollback du schéma initial supprime les parties et leurs données : arrêter les services applicatifs et sauvegarder la base avant toute intervention de ce type.

Pour des migrations ou un seed modifiés dans le dépôt, reconstruire d’abord l’image avec `docker compose build backend migrate seed`, car les fichiers SQL y sont embarqués.

## Partie avec deux navigateurs

Ce parcours sert de vérification manuelle sur deux profils ou appareils distincts.

1. Dans le navigateur A, ouvrir `/box/PB001`, saisir un pseudo et créer une partie. Ce joueur devient l’hôte.
2. Dans le navigateur B, un autre profil ou une fenêtre privée, ouvrir exactement la même URL et rejoindre avec un autre pseudo. Deux onglets ordinaires du même profil partagent le stockage et représentent donc le même joueur.
3. Le lobby affiche les deux joueurs. Seul l’hôte peut démarrer, à partir de deux joueurs.
4. Démarrer depuis A. Chaque écran affiche sa propre mission, son score et le classement.
5. Accomplir la mission dans la soirée puis appuyer sur **J’ai réussi**. Le serveur attribue les points prévus et une nouvelle mission.
6. Après chaque troisième completion réussie, noter facultativement l’ancienne mission avec **Bof**, **Ça va** ou **Fun**, ou choisir **Passer**. La nouvelle mission est déjà disponible.
7. Le classement de B se rafraîchit quasiment immédiatement via WebSocket. Sur smartphone, l’onglet **Classement** permet de le consulter.
8. Recharger la page : le token local restaure le même joueur, sa mission et son score.
9. L’hôte termine la partie via le bouton dédié et sa confirmation. Les deux écrans affichent le classement final et le nombre de missions accomplies.
10. **Revenir à l’accueil de la box** libère la session locale de cette partie et permet d’en créer ou rejoindre une nouvelle ; le pseudo reste mémorisé.

En mode Chaos, un événement est sélectionné après trois validations globales. `DOUBLE TROUBLE` double les trois validations suivantes, `BOUNTY` ajoute 100 points à la prochaine validation d’une cible choisie côté serveur, et `MISSION SHUFFLE` annule les missions courantes avant d’en attribuer de nouvelles à tout le monde.

Pour utiliser des téléphones : être sur le même Wi-Fi, définir `FRONTEND_URL=http://<IP_LAN_DU_PC>:3000`, garder `PUBLIC_API_URL=/api`, puis ouvrir `http://<IP_LAN_DU_PC>:3000/box/PB001`. Utiliser cette adresse sur tous les appareils ; `localhost` désigne le téléphone lui-même. Le port 3000 doit être autorisé par le pare-feu local. Le bouton de copie affiche le lien en texte si le presse-papiers est indisponible en HTTP.

### Accès téléphone via zrok

Un seul tunnel public vers `http://localhost:3000` suffit. Le frontend transmet déjà les appels `/api` au backend sur le réseau Docker : il n’est pas nécessaire d’exposer Go avec un second tunnel.

Pour une URL publique `https://<partage>.share.zrok.io`, utiliser dans `.env` :

```dotenv
FRONTEND_URL=https://<partage>.share.zrok.io
PUBLIC_API_URL=/api
API_INTERNAL_URL=http://backend:8080
```

Ne pas ajouter `:3000` ou `:8080` à l’URL publique HTTPS du tunnel. Ces ports concernent les services locaux. `API_INTERNAL_URL` reste l’adresse interne Docker, même si les joueurs passent par zrok.

Après modification de `.env`, appliquer les variables aux conteneurs :

```sh
docker compose up -d --no-deps backend frontend
```

Un simple `docker compose restart` ne recharge pas les variables d’environnement. Ouvrir ensuite `https://<partage>.share.zrok.io/box/PB001` sur le téléphone. Si l’URL temporaire du tunnel change, mettre `FRONTEND_URL` à jour et relancer la commande ci-dessus. Les sessions locales appartiennent à l’origine du navigateur : changer de domaine ne transfère pas le token joueur.

## PartyBox physique et mini-jeu de réaction

La migration `008_esp32_reaction` ajoute une identité device par box, une file de commandes HTTP, un planning persistant par partie et les challenges solo/duel. Le token ESP32 est distinct des sessions joueur et seul son hash SHA-256 est stocké. Le scheduler ne déclenche que pendant `playing`, avec un heartbeat récent et au maximum un challenge actif ; une panne du boîtier ne bloque jamais les missions web.

Le backend choisit le délai rouge → vert et l’ESP32 l’exécute localement. Le serveur traduit S2/S3 vers les joueurs, calcule le score, déduplique `event_id`, verrouille la partie et persiste le score avec `reaction_challenge_resolved` dans une seule transaction. Ce chemin n’appelle jamais la completion de mission et ne déclenche donc aucun effet Chaos.

Configuration serveur :

| Variable | Défaut | Rôle |
| --- | --- | --- |
| `REACTION_ENABLED` | `true` | Active le scheduler physique |
| `REACTION_MIN_INTERVAL_SECONDS` | `45` | Borne basse du prochain challenge |
| `REACTION_MAX_INTERVAL_SECONDS` | `90` | Borne haute, strictement supérieure à la borne basse |
| `REACTION_ASSIGNMENT_TIMEOUT_SECONDS` | `45` | Temps donné à l’hôte pour assigner S2/S3 |
| `REACTION_RESULT_TIMEOUT_SECONDS` | `15` | Marge de retour après le délai local |
| `DEVICE_ONLINE_TIMEOUT_SECONDS` | `10` | Âge maximal du dernier heartbeat |

Provisionner `PB001` après les migrations avec `docker compose run --rm --entrypoint /app/provision-device backend -box-id PB001`, puis conserver immédiatement le token affiché. Les secrets, la procédure ESP-Prog, le diagnostic prudent du PCB et le test complet sont détaillés dans [`firmware/README.md`](firmware/README.md).

## API REST

Toutes les réponses applicatives sont JSON. Les routes privées exigent `Authorization: Bearer <token>`. Les tokens ne sont jamais inclus dans le lobby, le classement ou les informations de box.

| Méthode / route                         | Accès                     | Corps ou résultat principal                                                                 |
| --------------------------------------- | ------------------------- | ------------------------------------------------------------------------------------------- |
| `GET /api/health`                       | Public                    | État de l’API et connexion à PostgreSQL                                                     |
| `GET /api/boxes/:boxId`                 | Public                    | Box et `active_game` ou `null`                                                              |
| `POST /api/boxes/:boxId/games`          | Public                    | `{ "name": "La soirée", "player_name": "Alice", "mode": "secret_missions" }` → session hôte |
| `POST /api/games/:gameId/join`          | Public                    | `{ "name": "Bob" }` → session joueur                                                        |
| `GET /api/games/:gameId`                | Joueur de la partie       | Statut, dates et joueurs avec scores                                                        |
| `GET /api/games/:gameId/chaos`          | Joueur d’une partie Chaos | Événement actif, progression et séquence                                                    |
| `POST /api/games/:gameId/start`         | Hôte                      | `{}` ; au moins deux joueurs                                                                |
| `POST /api/games/:gameId/end`           | Hôte                      | `{}` ; fige les scores, peut aussi fermer un lobby                                          |
| `POST /api/games/:gameId/ws-ticket`     | Joueur de la partie       | Ticket opaque à usage unique, valable 30 secondes                                           |
| `GET /api/games/:gameId/ws?ticket=…`    | Ticket court              | Connexion WebSocket limitée à la partie du joueur                                           |
| `GET /api/players/me`                   | Joueur                    | Identité, score, nombre de missions accomplies                                              |
| `GET /api/players/me/mission`           | Joueur                    | `{ "mission": ... }` ou `null` avant démarrage                                              |
| `POST /api/players/me/mission/complete` | Joueur                    | `{ "assignment_id": "<UUID de l’attribution>" }`                                            |
| `PUT /api/players/me/missions/:assignmentId/feedback` | Propriétaire de l’assignment terminé | `{ "rating": -1 }`, `0` ou `1` ; crée ou remplace l’avis |
| `GET /api/games/:gameId/leaderboard`    | Joueur de la partie       | `{ "players": [...] }`, par score décroissant                                               |
| `GET /api/games/:gameId/ai-missions/status` | Joueur de la partie | Disponibilité, taille du catalogue et générations restantes                                  |
| `POST /api/games/:gameId/ai-missions/generate` | Hôte dans le lobby | Ambiance, intensité, contexte et nombre ; retourne uniquement le total généré                 |
| `POST /api/boxes/:boxId/events`         | Contrat réservé           | `{ "type": "button_press" }` → **501 Not Implemented**, aucun effet                         |
| `GET /api/games/:gameId/reaction` | Joueur de la partie | État REST du boîtier et dernier challenge |
| `POST /api/games/:gameId/reaction/:challengeId/assign` | Hôte | Attribue un joueur à S2/S3 et crée la commande |
| `POST /api/device/boxes/:boxId/heartbeat` | Token device | Présence, version firmware, uptime et RSSI |
| `GET /api/device/boxes/:boxId/commands` | Token device | Commandes `pending`/`acknowledged` non expirées de cette box |
| `POST /api/device/boxes/:boxId/commands/:commandId/ack` | Token device | Ack retry-safe et passage du challenge à `armed` |
| `POST /api/device/boxes/:boxId/reaction-results` | Token device | Résultat terminal idempotent ; score calculé côté serveur |

Création et entrée renvoient `201` avec `{ token, player, game }`. Une validation renvoie `{ awarded_points, already_completed, player, mission }`. Le champ `mission.id` est l’identifiant de **l’attribution**, tandis que `mission.mission_id` identifie la mission du catalogue.

Erreurs : `{ "error": { "code": "conflict", "message": "..." } }`. Codes HTTP principaux : `400` données invalides, `401` session invalide, `403` action interdite, `404` introuvable, `409` conflit d’état ou pseudo déjà utilisé, `501` contrat ESP32 non implémenté.

### Temps réel et authentification WebSocket

Le navigateur demande d’abord un ticket avec son header `Authorization: Bearer <player-token>`. Le backend conserve uniquement le hash de ce ticket en mémoire, lié au joueur et à sa partie. Le ticket expire après 30 secondes et est supprimé dès sa première présentation à la route WebSocket. Le token joueur longue durée n’apparaît donc jamais dans l’URL.

Les connexions sont regroupées en mémoire par `gameId`. Les événements `player_joined`, `game_started`, `mission_completed` et `game_ended` contiennent seulement leur type, la partie, éventuellement le joueur concerné et l’heure. Aucun texte ou identifiant de mission n’est diffusé. À la réception, chaque client regroupe les notifications rapprochées et recharge son état et sa mission privée via les endpoints REST authentifiés.

La route WebSocket est explicitement exclue du timeout HTTP de 10 secondes ; les routes REST le conservent. Le serveur envoie des pings, retire les clients déconnectés et ferme les connexions au shutdown. Le client se reconnecte avec un nouveau ticket après 1, 2, 4, 8 secondes, puis jusqu’à un plafond de 30 secondes. Une resynchronisation REST à l’ouverture du socket couvre les événements survenus pendant une coupure.

Cette couche est volontairement locale au processus Go : une seule instance backend doit traiter une partie. Un déploiement multi-instance nécessiterait ultérieurement un bus inter-instance, hors périmètre du MVP.

Le serveur Node relaie l’upgrade de `/api/games/:gameId/ws` vers Go et Vite fait de même en développement. Caddy relaie nativement les WebSockets avec `reverse_proxy`, donc aucune configuration d’upgrade spécifique n’est nécessaire.

### Journal d’événements métier persistant

La migration `003_game_events` ajoute la table interne `game_events` : UUID, partie obligatoire, joueur optionnel, type, payload JSON et date de création. Les index `(game_id, created_at)` et `type` couvrent la lecture chronologique d’une partie et les futures analyses par catégorie. La suppression d’une partie efface son historique ; la suppression d’un joueur conserve les événements avec un `player_id` nul.

Ce journal PostgreSQL est distinct des notifications WebSocket : `models.GameEvent` conserve un historique pour le debug, l’analytics et de futurs traitements, tandis que `realtime.Event` reste un signal éphémère demandant au navigateur de relire REST. Aucune API publique n’expose actuellement `game_events`.

Les mutations existantes enregistrent `player_joined`, `game_started`, `mission_completed`, `game_ended` et `ai_missions_generated`. Chaos ajoute `chaos_event_triggered`, `chaos_event_consumed` et `mission_cancelled`. Un avis créé ou modifié ajoute `mission_feedback_submitted` avec `assignment_id` et `rating` ; répéter exactement le même avis ne duplique pas l’événement. La validation stocke `assignment_id` et le nombre réel de points accordés dans son payload. Chaque événement est écrit dans la même transaction que sa mutation. L’événement IA conserve uniquement le nombre, le mode, l’ambiance et l’intensité : aucune clé, réponse brute ou contexte libre. La constante générique `mission_assigned` est réservée, mais cet événement n’est pas encore écrit afin de ne pas complexifier le mécanisme d’attribution actuel.

### Moteur Chaos

Chaque partie Chaos possède une ligne `chaos_states` en PostgreSQL. Elle conserve l’événement courant, sa cible éventuelle, ses utilisations restantes, le nombre de validations depuis le dernier déclenchement et un numéro de séquence. PostgreSQL reste ainsi la source de vérité, y compris après un redémarrage du backend.

Le seuil de trois validations est centralisé dans le moteur. La sélection de l’événement et de la cible passe par une interface injectable : le runtime utilise une sélection aléatoire, tandis que les tests forcent chaque scénario. Un effet à plusieurs utilisations bloque tout événement incompatible jusqu’à sa consommation. Le score, l’état Chaos, les assignments et le journal métier sont validés ensemble dans une transaction.

`MISSION SHUFFLE` fait passer chaque assignment courant à `cancelled`, puis recrée exactement une mission `assigned` par joueur. Le nombre de missions accomplies ne compte que le statut `completed`. Aucun timer, scheduler, état Chaos côté navigateur ou nouveau payload WebSocket n’est utilisé.

### Génération de missions par IA

Dans le lobby, l’hôte peut générer un catalogue propre à sa partie pour `secret_missions`, `treasure_hunt` ou `chaos`. Pour Chaos, l’IA crée uniquement les missions de base ; les événements et leurs effets restent calculés par le moteur du backend. Le backend dérive le mode et le nombre de joueurs, limite le contexte libre à 300 caractères, appelle uniquement l’API OpenAI Responses avec Structured Outputs et demande explicitement `store=false`. La réponse est bornée et validée en Go avant toute écriture.

Une partie utilise exclusivement son catalogue IA lorsqu’il existe ; sinon elle garde les missions seed du même mode. Une régénération remplace l’ancien catalogue dans une transaction, avec l’événement métier correspondant. L’ancien catalogue reste intact si le fournisseur ou la validation échoue.

Une seule génération peut être active par partie et les transitions Start/End sont refusées pendant l’appel. Chaque partie dispose de cinq tentatives de génération par processus backend ; les générations réussies sont également comptées dans `game_events`, ce qui conserve la limite après un redémarrage. Les missions et attributions existantes restent utilisables une fois la limite atteinte.

### Preuve photo Treasure Hunt

Activer la vision dans `.env` avec une clé backend et un modèle compatible avec les images et Structured Outputs :

```dotenv
OPENAI_API_KEY=...
OPENAI_VISION_MODEL=...
OPENAI_VISION_DETAIL=low
```

`OPENAI_VISION_MODEL` est volontairement indépendant de `OPENAI_MODEL` : aucun fallback automatique. Sans clé ou sans modèle vision, Treasure Hunt conserve **J’ai trouvé** et sa validation manuelle. Secret Missions, Chaos et la génération de missions conservent leurs règles. `OPENAI_VISION_DETAIL` accepte uniquement `low` (défaut) ou `high`. Aucun modèle n’est codé en dur. Après modification, lancer `docker compose up --build -d`. Pour servir le build Node hors Docker, utiliser aussi `BODY_SIZE_LIMIT=6M`.

Le serveur indique les capacités avec `GET /api/players/me/mission/proof` (authentifié) : `available`, `assignment_id`, `attempts`, `max_attempts`, `remaining_attempts` et `accepted`. Le frontend affiche alors **Prendre une photo** et une sélection depuis la galerie, compresse en JPEG à 1280 px maximum / qualité 0,8, montre un aperçu puis attend l’envoi explicite du joueur. La caméra intégrée utilise `getUserMedia` et nécessite une origine sécurisée : `https://` sur téléphone (`localhost` est la seule exception HTTP). Un accès par IP LAN en `http://` peut utiliser la galerie, mais le navigateur y bloque la caméra intégrée ; utiliser le tunnel HTTPS décrit plus haut pour tester la capture sur téléphone.

`POST /api/players/me/mission/proof` accepte un corps `multipart/form-data` contenant exactement `assignment_id` et `image`. Le navigateur définit lui-même la boundary multipart. Le backend contrôle l’identité, l’appartenance et le statut de l’attribution, la partie en cours et le mode Treasure Hunt avant tout appel au fournisseur. Il reconnaît et décode les fichiers JPEG, PNG et WebP, avec un maximum de 5 Mio et 20 mégapixels. Le nom et le type MIME déclarés ne servent pas de preuve du format réel.

La couche `internal/vision` expose `Validator.Validate(ctx, ValidationRequest)` ; `internal/ai` reste consacré à la génération de missions. Le fournisseur vision envoie la mission exacte et une image à Responses API avec `store=false` et une sortie structurée, puis vérifie de nouveau le résultat en Go :

```json
{
  "verdict": "valid",
  "confidence": 0.94,
  "reason": "Un objet rouge est visible."
}
```

`verdict` vaut `valid`, `invalid` ou `uncertain`, `confidence` est comprise entre 0 et 1 et `reason` contient au plus 300 caractères. Le prompt demande uniquement une preuve visuelle raisonnable, sans identification de personne, reconnaissance faciale ni inférence sensible. Les objectifs impossibles à établir visuellement doivent rester `uncertain`.

Une transaction courte réserve la tentative, puis se ferme **avant** l’appel vision. Après l’analyse, une autre transaction enregistre le verdict et `mission_proof_evaluated` (assignment, verdict et confiance seulement). Si le verdict est `valid`, le service appelle ensuite `Service.Complete` : les verrous, le score, la nouvelle mission, l’idempotence et l’événement `mission_completed` restent gérés par le flux existant. La route de completion manuelle exige elle aussi une preuve acceptée lorsque la vision est configurée. Une réponse perdue se rejoue sans nouvel appel IA et sans double score. Si la partie se termine pendant l’analyse, la preuve est conservée mais la completion est refusée.

La réponse HTTP 200 contient `proof` (id, assignment, verdict, confiance, raison, date) et, uniquement après acceptation, `completion` avec les points réellement accordés et la nouvelle mission. `invalid` affiche **Pas encore**, `uncertain` affiche **Difficile à vérifier** ; aucun des deux ne donne de points. Le WebSocket existant ne diffuse que `mission_completed` après une completion réussie. Ni photo, ni verdict, ni raison ne sont transmis aux autres joueurs.

La migration `006_treasure_photo_validation` ajoute `mission_proofs`, indexée par attribution et date, et `player_missions.proof_attempts`. Les contraintes SQL bornent les verdicts, la confiance et la raison. Le rollback supprime les preuves et compteurs sans modifier les scores ni les attributions ; sauvegarder cet historique avant un rollback.

La limite est **cinq appels par attribution**, définie une seule fois dans `services.MaxProofAttempts`. Les réservations persistent, y compris après une erreur réseau, une réponse fournisseur invalide ou un redémarrage. Les requêtes concurrentes partagent cette limite. Les fichiers ou attributions refusés avant l’appel ne consomment rien ; à la limite, le serveur répond `429 proof_limit`. Les erreurs du fournisseur répondent `503 vision_unavailable`, sans détail sensible. Aucun retry automatique payant n’est effectué. Une mission ayant épuisé ses tentatives reste bloquée dans cette V1 ; il n’existe pas encore de passage à la mission suivante ni de dérogation hôte.

PartyBox ne conserve aucune photo sur disque, en BDD, dans les logs ou dans le stockage du navigateur. Le multipart est lu en mémoire sans fichier temporaire ; les URLs d’aperçu sont révoquées, et le JPEG recompressé dans le navigateur ne garde pas les métadonnées EXIF. Les images sont transmises au fournisseur pour l’analyse : les règles de conservation du fournisseur restent distinctes de celles de PartyBox. Seuls les verdicts, raisons et compteurs sont persistés localement.

Le coût dépend du modèle configuré, de l’image et du niveau de détail. La compression, `low` par défaut, la sortie bornée et la limite de cinq tentatives maîtrisent le volume des requêtes ; aucun prix fixe ni garantie de précision n’est supposé. Références de l’implémentation : [images dans Responses API](https://developers.openai.com/api/docs/guides/images-vision) et [Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs).

### Identité et cohérence

- Token opaque de **32 octets aléatoires**, généré avec `crypto/rand`, encodé en base64url ; seul son **hash SHA-256** est conservé dans `players.token_hash`.
- Le token est stocké dans `localStorage` par box, avec l’identifiant de partie. Le pseudo n’identifie jamais un joueur. Les sessions restent valables pour consulter une partie terminée.
- Un pseudo est limité à 24 caractères, un nom de partie à 60 ; les pseudos sont uniques dans une partie, sans tenir compte de la casse. Les caractères de contrôle sont refusés.
- L’hôte est désigné par `players.is_host`, avec un index unique partiel. Aucun `host_player_id` redondant n’est nécessaire.
- Une seule partie active par box et une seule mission courante par joueur sont garanties par des index SQL.
- Création et entrée sont transactionnelles. Démarrage, validation, entrée et fin verrouillent la partie concernée pour sérialiser les changements ; leur événement métier est enregistré dans la même transaction.
- La validation, les points, la nouvelle attribution et l’événement persistant sont atomiques. Répéter le même `assignment_id` ne donne aucun point supplémentaire et ne duplique pas l’événement. Le client ne choisit jamais le score accordé.
- Un joueur peut rejoindre une partie déjà commencée : il reçoit aussitôt une mission. Une partie terminée refuse les nouveaux joueurs et toute nouvelle validation.
- Les missions des autres joueurs ne sont pas exposées par l’API. Les égalités de score sont affichées au même rang ; l’ordre visuel est stabilisé par la date d’arrivée puis l’identifiant.

## Développement natif (optionnel)

Prérequis supplémentaires : Go 1.26+, Node.js 24 LTS, npm et PostgreSQL 17 avec une base vide accessible. Les versions JavaScript exactes sont figées dans `package-lock.json` et les dépendances Go dans `go.mod` / `go.sum`.

Dans un terminal :

```sh
cd backend
export DATABASE_URL='postgres://partybox:partybox_dev_only@localhost:5432/partybox?sslmode=disable'
export BACKEND_PORT=8080
export FRONTEND_URL='http://localhost:5173'
go run ./cmd/api migrate
go run ./cmd/api seed
go run ./cmd/api
```

Dans un deuxième terminal, depuis la racine :

```sh
cd frontend
npm ci
API_INTERNAL_URL=http://localhost:8080 PUBLIC_API_URL=/api npm run dev
```

Ouvrir `http://localhost:5173/box/PB001`. Pour utiliser PostgreSQL depuis Compose avec les serveurs Go et SvelteKit natifs, lancer d’abord `docker compose up -d postgres` à la racine. La base est accessible sur `127.0.0.1:5432` (ou le port choisi dans `POSTGRES_PORT`). Adapter l’URL de connexion du backend natif si le port ou les identifiants changent.

### Vérifications et tests

Le package realtime contient des tests avec `httptest` et de vrais clients WebSocket. Les tests d’intégration PostgreSQL utilisent un schéma isolé et couvrent les trois modes, les migrations, les catalogues IA, Chaos et le feedback. Le feedback vérifie la propriété et le statut de l’assignment, les trois notes, les entrées invalides, l’upsert et l’événement métier.

```sh
cd backend
go test ./...
go vet ./...
```

Les tests PostgreSQL sont ignorés si `TEST_DATABASE_URL` n’est pas défini. Ils couvrent aussi l’auth device, le heartbeat sans spam, le scheduler et ses expirations, l’assignation host-only, le polling/ack, les résultats concurrents, le barème et l’isolation de Chaos. Pour exécuter tous les scénarios sur un serveur local, avec un compte autorisé à créer des schémas :

```sh
TEST_DATABASE_URL='postgres://partybox:partybox_dev_only@127.0.0.1:5432/partybox?sslmode=disable' go test ./...
```

Les tests de preuve photo utilisent un fake `Validator` et un serveur HTTP local simulant Responses : aucun appel réel OpenAI. Ils couvrent les trois verdicts, les refus avant appel, les fichiers invalides/trop gros, les erreurs fournisseur, la persistance, les quotas concurrents et après redémarrage, la completion idempotente, le WebSocket, la fin de partie pendant l’analyse, le fallback manuel et la migration `005 → 006` avec rollback.

```sh
cd frontend
npm test
npm run check
npm run build
```

```sh
pio test -d firmware -e native
pio run -d firmware -e esp32s2
```

```sh
python -m compileall ml
```

Depuis la racine : `docker compose config --quiet` et `docker compose up --build`. Les healthchecks Compose vérifient la disponibilité des services. Les parcours création → arrivée d’un second joueur → démarrage → validation → classement → fin sont vérifiables pour `secret_missions`, `treasure_hunt` et `chaos`, avec les quatre notifications WebSocket associées.

## VPS Linux et Caddy

Une configuration de déploiement est préparée, sans déploiement public effectué. Sur un VPS disposant de Docker et Compose :

1. Cloner le dépôt et créer `.env`. Choisir un mot de passe PostgreSQL propre à ce serveur et le reporter dans `DATABASE_URL`.
2. Faire pointer le DNS de `DOMAIN` vers le VPS. Renseigner un véritable `ACME_EMAIL`.
3. Définir `FRONTEND_URL=https://<domaine>`, `PUBLIC_API_URL=/api` et `FRONTEND_BIND=127.0.0.1`.
4. Ouvrir les ports 80 et 443, puis lancer :

```sh
docker compose --profile production up --build -d
```

Caddy termine HTTPS, conserve ses certificats dans un volume et transmet au frontend. Go reste dans le réseau privé. Les ports publiés de PostgreSQL et du frontend sont limités à la boucle locale du VPS. Pour administrer la base depuis un autre ordinateur, utiliser un tunnel SSH vers son port local. Prévoir les sauvegardes PostgreSQL et leur restauration avant une utilisation réelle. `sslmode=disable` concerne les connexions locales de cette configuration ; une base distante doit utiliser TLS.

La box `PB001` reste une référence de développement. Pour provisionner une autre box, l’insérer en base avec un identifiant unique, puis programmer le tag NFC avec `/box/<identifiant>`. Le MVP ne contient pas d’interface d’administration des boxes.

## Limites et prochaines étapes

- La réussite reste déclarative en Secret Missions, Chaos et Treasure Hunt sans vision. La preuve photo optionnelle de Treasure Hunt est une vérification visuelle, pas une garantie antitriche ; les missions abstraites ou historiques peuvent rester incertaines.
- Pas de transfert d’hôte, de récupération de token perdu, de révocation, d’expiration automatique ou de nettoyage des anciennes parties. Si l’hôte perd son stockage local pendant une partie, une intervention en base sera nécessaire pour la fermer.
- Pas de présence connectée/déconnectée : le lobby liste les inscrits. Pas de fonctionnement hors ligne ; une connexion au serveur est nécessaire.
- Les événements Chaos sont déclenchés par le nombre de validations, sans timer. Ils ne comprennent pour l’instant que Double Trouble, Bounty et Mission Shuffle.
- Pas encore de PWA installable, service worker, push, compte email/OAuth, galerie de photos, MQTT, Redis ou modèle ML fourni en production. Le pipeline ML peut entraîner un ranker global sur les données notées réelles et l’intégration Go reste inactive sans artifact explicitement configuré.
- Le parcours realtime est couvert au niveau transport/hub et dans deux contextes Chromium. Le firmware ESP32-S2 et sa logique native compilent en CI, mais le PCB physique, les GPIO inconnus, l’ESP-Prog et le parcours radio réel doivent encore être validés sur place.
- Le tag NFC reste passif et doit être programmé séparément avec `https://<domain>/box/PB001`. Il n’existe volontairement ni OTA, MQTT, Bluetooth, lecteur NFC actif, logique batterie, PIR ou WebSocket ESP32.

Références techniques : [serveur Node SvelteKit](https://svelte.dev/docs/kit/adapter-node), [Gin](https://gin-gonic.com/en/docs/quickstart/), [pgx v5](https://pkg.go.dev/github.com/jackc/pgx/v5).
