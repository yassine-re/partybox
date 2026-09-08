# PartyBox

PartyBox transforme une soirée IRL en jeu de missions secrètes. Un tag NFC passif dans le boîtier ouvre une URL comme `https://partybox.example.com/box/PB001`. Chaque joueur utilise son téléphone ; Go et PostgreSQL centralisent les parties, missions et scores sur un serveur.

Ce dépôt contient le premier MVP logiciel : création de partie, lobby partagé, démarrage par l’hôte, missions privées, validation virtuelle, points, nouvelle mission et classement final. Le firmware et le machine learning ont uniquement leur emplacement préparé.

## Démarrage rapide

Prérequis : Git, Docker Engine avec Docker Compose v2+ (ou Docker Desktop démarré). Aucun Go, Node ou PostgreSQL local n’est nécessaire avec Compose.

Depuis la racine du dépôt :

```sh
cp .env.example .env
docker compose up --build
```

Ouvrir **[http://localhost:3000/box/PB001](http://localhost:3000/box/PB001)**.

Compose attend PostgreSQL, applique les migrations, exécute le seed idempotent, puis démarre l’API et le frontend. La box `PB001` et **18 missions** sont créées automatiquement. Aucune partie ni joueur de démonstration n’est créé.

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
       │ fetch /api → passerelle serveur de même origine
       ▼
Go + Gin :8080
       │ pgx / transactions SQL
       ▼
PostgreSQL :5432
```

- **Frontend** : SvelteKit 2, Svelte 5, TypeScript, CSS simple et composants réutilisables. Interface mobile sombre, accents citron et rose. Aucun framework CSS ni gestionnaire d’état supplémentaire. Le build utilise `adapter-node` pour le déploiement sur VPS.
- **Backend** : les handlers gèrent HTTP/JSON, les services gèrent les règles et l’enchaînement transactionnel, les repositories contiennent le SQL et les verrous PostgreSQL. Pas d’ORM.
- **Communication** : `fetch` regroupé dans `frontend/src/lib/api/`. Polling toutes les trois secondes quand la page est visible, bouton de rafraîchissement et resynchronisation au retour dans l’onglet. Aucun WebSocket.
- **Réseau** : le navigateur appelle `/api` sur son propre domaine. La passerelle SvelteKit relaie vers `backend` sur le réseau privé Compose. Le frontend est accessible sur le réseau de développement ; PostgreSQL publie uniquement un port local sur `127.0.0.1` pour les explorateurs de base de données. L’API Go ne publie pas de port sur l’hôte.
- **Persistance** : migrations SQL versionnées, seed idempotent et volume PostgreSQL. Les conteneurs API et frontend tournent avec des utilisateurs sans privilèges.

Le dossier courant est directement la racine PartyBox, même s’il porte un autre nom localement ; aucun sous-dossier `partybox/` supplémentaire n’est nécessaire.

## Arborescence

```text
.
├── backend/
│   ├── cmd/api/main.go          # Serveur et commandes migrate / seed
│   ├── internal/
│   │   ├── database/            # Pool, exécution des migrations et du seed
│   │   ├── handlers/            # Routes Gin, auth Bearer, JSON et erreurs
│   │   ├── services/            # Identités et cycle de vie d’une partie
│   │   ├── repositories/        # SQL PostgreSQL et transactions
│   │   └── models/              # Types métier / réponses JSON
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/lib/api/             # Client fetch, types et sessions locales
│   ├── src/lib/components/      # Entrée, jeu, mission et liste/classement
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
│   └── seed.sql
├── firmware/
│   ├── platformio.ini
│   ├── src/main.cpp
│   └── include/
├── ml/                         # Réservé à Python, aucune implémentation ML
├── infra/Caddyfile
├── docker-compose.yml
├── .env.example
├── .gitignore
└── README.md
```

Le lobby, le jeu, le classement et la fin partagent `/box/[boxId]` : le même lien NFC reste pertinent à chaque étape. Les composants sont séparés sans multiplier les routes ni les identifiants dans les liens partagés.

## Configuration

`.env` est ignoré par Git. Les valeurs livrées sont uniquement destinées au développement. Ne jamais mettre un secret dans une variable `PUBLIC_*`, car elle est accessible au navigateur.

| Variable | Valeur de développement | Rôle |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://partybox:partybox_dev_only@postgres:5432/partybox?sslmode=disable` | Connexion pgx depuis Docker |
| `POSTGRES_DB` | `partybox` | Base créée au premier démarrage PostgreSQL |
| `POSTGRES_USER` | `partybox` | Utilisateur PostgreSQL |
| `POSTGRES_PORT` | `5432` | Port PostgreSQL publié uniquement sur `127.0.0.1` |
| `POSTGRES_PASSWORD` | `partybox_dev_only` | Mot de passe de développement |
| `BACKEND_PORT` | `8080` | Port interne Go |
| `FRONTEND_PORT` | `3000` | Port du frontend publié sur l’hôte |
| `FRONTEND_BIND` | `0.0.0.0` | Adresse d’écoute sur l’hôte ; utiliser `127.0.0.1` derrière Caddy en production |
| `FRONTEND_URL` | `http://localhost:3000` | Origine publique pour SvelteKit et CORS de Go |
| `PUBLIC_API_URL` | `/api` | Base d’URL pour le navigateur, configurée au démarrage |
| `API_INTERNAL_URL` | `http://backend:8080` | Backend joint par le serveur SvelteKit |
| `DOMAIN` | `partybox.example.com` | Domaine du profil Caddy |
| `ACME_EMAIL` | `you@example.com` | Contact pour les certificats HTTPS |

Maintenir `DATABASE_URL` cohérent avec les trois variables PostgreSQL. Si le mot de passe contient des caractères réservés aux URL, les encoder dans `DATABASE_URL`. Maintenir également `API_INTERNAL_URL` cohérent avec `BACKEND_PORT` et `FRONTEND_URL` avec l’URL utilisée par les joueurs.

Changer `POSTGRES_PASSWORD` dans `.env` ne modifie pas le mot de passe d’une base déjà initialisée : effectuer sa rotation dans PostgreSQL aussi.

Le binaire Go accepte également `MIGRATIONS_DIR` et `SEED_FILE`, configurées dans son image Docker. SvelteKit utilise `HOST=0.0.0.0`, `PORT=3000` et `BODY_SIZE_LIMIT=16K` dans le conteneur. En lancement natif, il faut charger les variables dans le shell ; le backend ne lit pas automatiquement `.env`.

## Connexion avec un explorateur de base de données

PostgreSQL est accessible depuis un explorateur installé sur le même ordinateur (DBeaver, DataGrip, TablePlus ou extension d’éditeur) avec ces paramètres :

| Paramètre | Valeur |
| --- | --- |
| Type de connexion | PostgreSQL |
| Hôte | `127.0.0.1` |
| Port | `5432` (ou `POSTGRES_PORT` dans `.env`) |
| Base | `partybox` (ou `POSTGRES_DB`) |
| Utilisateur | `partybox` (ou `POSTGRES_USER`) |
| Mot de passe | La valeur de `POSTGRES_PASSWORD` dans `.env` |
| SSL | Désactivé pour cette connexion locale |
| Schéma | `public` |

Le nom d’hôte `postgres` de `DATABASE_URL` est réservé au réseau Docker. Depuis un explorateur local, utiliser `127.0.0.1`, sans URL zrok ni préfixe HTTP. La publication du port est limitée à la boucle locale : elle n’expose pas la base sur le Wi-Fi ou Internet.

Si le port 5432 est déjà occupé, définir par exemple `POSTGRES_PORT=5433`, puis appliquer avec `docker compose up -d postgres`. Utiliser alors 5433 dans l’explorateur ; `DATABASE_URL` du backend conserve `postgres:5432`, le port interne Docker.

## Migrations et seed

Les deux opérations sont automatiques avec le démarrage Compose. Pour les relancer explicitement sur une stack déjà démarrée :

```sh
docker compose run --rm migrate
docker compose run --rm seed
```

`migrate` applique les fichiers `*.up.sql` par ordre lexical et inscrit leurs versions dans `schema_migrations`. Les migrations et le suivi des versions partagent une transaction ; un verrou évite deux exécutions simultanées. Ajouter une nouvelle migration numérotée plutôt que modifier une migration déjà appliquée.

Le seed insère `PB001` et 18 missions avec points, catégorie et difficulté (1 à 3), sans dupliquer les lignes ni remplacer les données existantes. Les missions déjà jouées sont évitées tant qu’il reste des missions inédites ; après épuisement du catalogue, la moins récemment attribuée revient.

Le fichier `.down.sql` est fourni pour un retour arrière manuel. Le binaire n’exécute pas de rollback automatique. Un rollback du schéma initial supprime les parties et leurs données : arrêter les services applicatifs et sauvegarder la base avant toute intervention de ce type.

Pour des migrations ou un seed modifiés dans le dépôt, reconstruire d’abord l’image avec `docker compose build backend migrate seed`, car les fichiers SQL y sont embarqués.

## Partie avec deux navigateurs

Ce parcours est documenté pour la prochaine vérification manuelle ; il n’a pas été exécuté à cette étape, conformément à la consigne de différer les tests.

1. Dans le navigateur A, ouvrir `/box/PB001`, saisir un pseudo et créer une partie. Ce joueur devient l’hôte.
2. Dans le navigateur B, un autre profil ou une fenêtre privée, ouvrir exactement la même URL et rejoindre avec un autre pseudo. Deux onglets ordinaires du même profil partagent le stockage et représentent donc le même joueur.
3. Le lobby affiche les deux joueurs. Seul l’hôte peut démarrer, à partir de deux joueurs.
4. Démarrer depuis A. Chaque écran affiche sa propre mission, son score et le classement.
5. Accomplir la mission dans la soirée puis appuyer sur **J’ai réussi**. Le serveur attribue les points prévus et une nouvelle mission.
6. Le classement de B se rafraîchit sous trois secondes. Sur smartphone, l’onglet **Classement** permet de le consulter.
7. Recharger la page : le token local restaure le même joueur, sa mission et son score.
8. L’hôte termine la partie via le bouton dédié et sa confirmation. Les deux écrans affichent le classement final et le nombre de missions accomplies.
9. **Revenir à l’accueil de la box** libère la session locale de cette partie et permet d’en créer ou rejoindre une nouvelle ; le pseudo reste mémorisé.

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

## API REST

Toutes les réponses applicatives sont JSON. Les routes privées exigent `Authorization: Bearer <token>`. Les tokens ne sont jamais inclus dans le lobby, le classement ou les informations de box.

| Méthode / route | Accès | Corps ou résultat principal |
| --- | --- | --- |
| `GET /api/health` | Public | État de l’API et connexion à PostgreSQL |
| `GET /api/boxes/:boxId` | Public | Box et `active_game` ou `null` |
| `POST /api/boxes/:boxId/games` | Public | `{ "name": "La soirée", "player_name": "Alice" }` → session hôte |
| `POST /api/games/:gameId/join` | Public | `{ "name": "Bob" }` → session joueur |
| `GET /api/games/:gameId` | Joueur de la partie | Statut, dates et joueurs avec scores |
| `POST /api/games/:gameId/start` | Hôte | `{}` ; au moins deux joueurs |
| `POST /api/games/:gameId/end` | Hôte | `{}` ; fige les scores, peut aussi fermer un lobby |
| `GET /api/players/me` | Joueur | Identité, score, nombre de missions accomplies |
| `GET /api/players/me/mission` | Joueur | `{ "mission": ... }` ou `null` avant démarrage |
| `POST /api/players/me/mission/complete` | Joueur | `{ "assignment_id": "<UUID de l’attribution>" }` |
| `GET /api/games/:gameId/leaderboard` | Joueur de la partie | `{ "players": [...] }`, par score décroissant |
| `POST /api/boxes/:boxId/events` | Contrat réservé | `{ "type": "button_press" }` → **501 Not Implemented**, aucun effet |

Création et entrée renvoient `201` avec `{ token, player, game }`. Une validation renvoie `{ awarded_points, already_completed, player, mission }`. Le champ `mission.id` est l’identifiant de **l’attribution**, tandis que `mission.mission_id` identifie la mission du catalogue.

Erreurs : `{ "error": { "code": "conflict", "message": "..." } }`. Codes HTTP principaux : `400` données invalides, `401` session invalide, `403` action interdite, `404` introuvable, `409` conflit d’état ou pseudo déjà utilisé, `501` contrat ESP32 non implémenté.

### Identité et cohérence

- Token opaque de **32 octets aléatoires**, généré avec `crypto/rand`, encodé en base64url ; seul son **hash SHA-256** est conservé dans `players.token_hash`.
- Le token est stocké dans `localStorage` par box, avec l’identifiant de partie. Le pseudo n’identifie jamais un joueur. Les sessions restent valables pour consulter une partie terminée.
- Un pseudo est limité à 24 caractères, un nom de partie à 60 ; les pseudos sont uniques dans une partie, sans tenir compte de la casse. Les caractères de contrôle sont refusés.
- L’hôte est désigné par `players.is_host`, avec un index unique partiel. Aucun `host_player_id` redondant n’est nécessaire.
- Une seule partie active par box et une seule mission courante par joueur sont garanties par des index SQL.
- Création et entrée sont transactionnelles. Démarrage, validation, entrée et fin verrouillent la partie concernée pour sérialiser les changements.
- La validation, les points et la nouvelle attribution sont atomiques. Répéter le même `assignment_id` ne donne aucun point supplémentaire. Le client ne choisit jamais le score accordé.
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

Les tests automatisés et la vérification fonctionnelle multi-navigateurs sont **reportés à la demande du porteur du projet**. Aucun fichier de test n’est ajouté. Les commandes suivantes servent uniquement à vérifier les types, la compilation et la configuration :

```sh
cd frontend
npm run check
npm run build
```

```sh
cd backend
go build ./...
```

Depuis la racine : `docker compose config --quiet` et `docker compose build`. Les healthchecks Compose vérifient uniquement la disponibilité des services ; ils ne valident pas la boucle métier.

Quand les tests seront autorisés, couvrir en priorité la boucle complète, les droits hôte, les requêtes de validation concurrentes, les collisions de création et la restauration de session.

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

- La réussite d’une mission repose sur la déclaration du joueur. Pas de validation matérielle ni d’antitriche.
- Pas de transfert d’hôte, de récupération de token perdu, de révocation, d’expiration automatique ou de nettoyage des anciennes parties. Si l’hôte perd son stockage local pendant une partie, une intervention en base sera nécessaire pour la fermer.
- Pas de présence connectée/déconnectée : le lobby liste les inscrits. Pas de fonctionnement hors ligne ; une connexion au serveur est nécessaire.
- Pas encore de PWA installable, service worker, push, WebSocket, compte email/OAuth, upload, MQTT, Redis ou ML. La structure SvelteKit et les assets statiques permettent d’ajouter une PWA plus tard.
- Les tests de la boucle fonctionnelle restent à réaliser. Le HTTPS public et le firmware n’ont pas été vérifiés sur du matériel réel.

Pour intégrer l’ESP32 ensuite :

1. Choisir la carte, câbler le bouton et implémenter l’antirebond dans le squelette PlatformIO / Arduino.
2. Provisionner `box_id`, Wi-Fi et une clé matérielle distincte des tokens joueurs, sans les committer.
3. Activer la route d’événements avec authentification du boîtier, identifiant d’événement et gestion des retries.
4. Définir comment l’appui désigne le joueur à valider, puis appeler la logique de validation transactionnelle existante.
5. Encoder l’URL HTTPS dans le tag NFC passif et vérifier le parcours sur plusieurs téléphones avec le matériel.

Références techniques : [serveur Node SvelteKit](https://svelte.dev/docs/kit/adapter-node), [Gin](https://gin-gonic.com/en/docs/quickstart/), [pgx v5](https://pkg.go.dev/github.com/jackc/pgx/v5).
