# Dataset de feedback PartyBox

Ce dossier prépare les données nécessaires à un futur système de recommandation. Il ne contient encore aucun modèle entraîné ou utilisé en production.

## Installation et export

```bash
python -m venv .venv
source .venv/bin/activate
pip install -r ml/requirements.txt
DATABASE_URL='postgresql://partybox:partybox_dev_only@127.0.0.1:5432/partybox' \
  python ml/export_dataset.py
```

Le CSV est écrit dans `ml/data/mission_feedback.csv`. Ce dossier est ignoré par Git : aucune donnée joueur réelle ne doit être commitée. Utiliser `--output chemin.csv` pour choisir une autre destination.

## Schéma supervisé

Une ligne correspond à un assignment terminé et noté. Les features actuelles sont :

- `mode`, `category`, `difficulty`, `points`, `source` ;
- `duration_seconds`, calculé depuis `assigned_at` et `completed_at` ;
- `game_size`, nombre de joueurs présents dans la partie au moment de la completion.

La cible est `rating` (`-1`, `0`, `1`). Un futur modèle binaire pourra dériver `liked = rating == 1`. Les identifiants et `completed_at` sont conservés pour contrôler les doublons et analyser les périodes, mais ne constituent pas nécessairement des features d’entraînement.

Le script exclut les completions sans feedback et les durées impossibles, vérifie les champs essentiels, puis affiche la couverture du feedback, la répartition des notes et des modes, ainsi que les moyennes par mode, catégorie et difficulté.

## Limites

PartyBox n’a pas encore d’identité persistante entre plusieurs parties. Le dataset permet d’évaluer les missions globalement, de comparer les catégories et les modes, puis de préparer un classement contextuel. Il ne permet pas encore une personnalisation historique fiable par personne. Il faut collecter un volume représentatif avant d’entraîner un modèle.
