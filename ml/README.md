# Ranking ML des missions PartyBox

Ce dossier entraîne hors ligne un premier modèle **global et contextuel** qui estime `P(liked = 1 | mission, partie)`. Il ne personnalise pas les missions par joueur : PartyBox ne possède pas encore d’identité utilisateur persistante entre plusieurs parties.

Le pipeline n’invente aucune donnée. Les fixtures synthétiques de `tests/` vérifient uniquement le code et ne sont jamais utilisées pour produire des métriques ou un modèle réel.

## Données et cible

`dataset.py` lit les assignments terminés qui possèdent un feedback dans `mission_feedback`. Une ligne conserve les identifiants pour l’audit et le split, mais seules les informations disponibles **avant** de jouer la mission alimentent le modèle :

- catégorielles : `mode`, `category`, `source` ;
- numériques : `difficulty`, `points`, `game_size`.

La cible binaire est :

```python
liked = (rating == 1).astype(int)
```

Une note `1` devient donc positive ; `0` et `-1` deviennent négatives. Cette simplification est propre à la V1. Un futur modèle ordinal pourra distinguer neutre et négatif.

Les colonnes suivantes sont explicitement interdites comme features afin d’éviter les fuites de données : `rating`, `liked`, `completed_at`, `duration_seconds`, `assignment_id`, `mission_id`, `player_id` et `game_id`. `duration_seconds` reste utilisable pour l’analyse descriptive, mais elle est inconnue au moment du ranking. Les IDs ne servent qu’à l’audit et `game_id` au split.

## Installation, export et entraînement

Depuis la racine :

```bash
python -m venv .venv
source .venv/bin/activate
pip install -r ml/requirements.txt

DATABASE_URL='postgresql://partybox:partybox_dev_only@127.0.0.1:5432/partybox' \
  python ml/export_dataset.py

DATABASE_URL='postgresql://partybox:partybox_dev_only@127.0.0.1:5432/partybox' \
  python ml/train.py
```

Le CSV d’audit est écrit dans `ml/data/mission_feedback.csv`. `ml/data/` et `ml/artifacts/` sont ignorés par Git afin qu’aucune donnée réelle ou modèle local ne soit commité.

L’entraînement est refusé si un seul de ces seuils centralisés dans `model_metadata.py` manque :

- 50 lignes notées ;
- 10 positives ;
- 10 négatives ;
- 5 parties distinctes.

Le message `Dataset insuffisant pour entraîner un modèle fiable` est alors affiché, avec chaque seuil manquant. Aucun artifact existant n’est créé ou remplacé.

## Modèle et évaluation

Le pipeline sauvegardé est :

```text
ColumnTransformer
├── OneHotEncoder(handle_unknown="ignore")
└── StandardScaler
↓
LogisticRegression
```

Le split 80/20 est déterministe et groupé par `game_id` : toutes les lignes d’une même partie restent du même côté. Le pipeline cherche un split où les deux classes sont présentes dans le train et, si les données le permettent, dans le test. Si le test ne contient qu’une classe, la ROC AUC est omise au lieu d’afficher une valeur trompeuse.

L’évaluation calcule accuracy, precision, recall, F1, ROC AUC lorsque définie, matrice de confusion et accuracy d’une baseline qui prédit toujours la classe majoritaire du train. Le modèle n’est marqué utilisable que si le test contient les deux classes et si son accuracy tenue à l’écart est strictement supérieure à cette baseline. Afficher le rapport déjà sauvegardé :

```bash
python ml/evaluate.py
```

Les coefficients les plus positifs et négatifs sont conservés comme **associations apprises dans le dataset**, jamais comme relations causales.

## Artifacts et prédiction

Un entraînement admissible écrit atomiquement :

- `ml/artifacts/mission_ranker.joblib` : pipeline Python complet ;
- `ml/artifacts/mission_ranker.metadata.json` : volumes, split, seuils, métriques et coefficients, sans nom ni identifiant joueur ;
- `ml/artifacts/mission_ranker.json` : intercept, catégories OneHot, coefficients et normalisation numérique portables vers Go.

Ne charger qu’un artifact `joblib` produit et contrôlé localement : ce format n’est pas sûr lorsqu’il provient d’une source non fiable.

`predict.py` expose :

```python
scores = score_missions(model, missions, {"game_size": 5})
ranked = rank_missions(model, missions, {"game_size": 5})
```

`score_missions` renvoie une probabilité par candidate. `rank_missions` renvoie des copies triées par `ml_score` décroissant, sans modifier `mission.points`. Les catégories inconnues sont tolérées et reçoivent une contribution OneHot nulle.

Test CLI avec un tableau JSON de candidates :

```bash
python ml/predict.py --missions /tmp/missions.json --game-size 5
```

## Intégration Go optionnelle

Le backend charge le JSON portable seulement si `ML_RANKER_MODEL` pointe vers un fichier valide **et** si les metadata portables indiquent que le modèle bat la baseline :

```bash
cd backend
ML_RANKER_MODEL="$PWD/../ml/artifacts/mission_ranker.json" \
DATABASE_URL='postgresql://partybox:partybox_dev_only@127.0.0.1:5432/partybox' \
  go run ./cmd/api
```

Sans variable, PartyBox garde silencieusement l’attribution aléatoire actuelle. Si un chemin est configuré mais que le fichier est absent, invalide ou sous la baseline, un avertissement est émis et le même fallback est conservé. Aucun microservice Python n’est nécessaire.

Avec un scorer actif, l’attribution conserve le catalogue du `GameMode`, la priorité aux missions inédites puis les moins récentes, le catalogue IA propre à la partie et l’interdiction de répétition immédiate. Elle charge jusqu’à dix candidates, les score, conserve le top 5 puis choisit aléatoirement dans ce top. Ainsi, le ML influence la sélection sans rendre toutes les parties déterministes.

Les fixtures JSON communes de `tests/fixtures/` vérifient que le calcul Go `sigmoid(intercept + Σ coefficients × features)` diffère du score Python de moins de `1e-6`.

## Tests

```bash
python -m pytest ml/tests
python -m compileall ml
cd backend && go test ./... && go vet ./...
```

## Limites et suite

- Un petit volume réel, des feedbacks facultatifs ou une classe dominante peuvent rendre le modèle inutilisable.
- Les avis collectés souffrent d’un biais de sélection : les joueurs qui notent ne représentent pas nécessairement tous les joueurs.
- Les coefficients décrivent des corrélations propres aux parties observées, pas des causes.
- Le ranking est global/contextuel, sans historique individuel fiable.
- Le modèle ne change ni les points ni les règles de jeu et n’est jamais requis pour démarrer PartyBox.

Évolution envisagée, après collecte suffisante et création volontaire d’une identité persistante :

```text
LLM génère des candidates
↓
ranker global puis préférences historiques par catégorie
↓
top missions compatibles
↓
Game Engine
```

Il faudra alors mesurer la qualité sur plusieurs périodes, surveiller la couverture des feedbacks et comparer proprement le ranking au mécanisme aléatoire avant de parler de personnalisation.
