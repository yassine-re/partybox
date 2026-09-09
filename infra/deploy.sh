#!/usr/bin/env bash
set -Eeuo pipefail

tag="${1:-}"
app_dir="${HOME}/partybox"

if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Tag invalide : $tag"
  exit 1
fi

cd "$app_dir"

if [[ ! -f .env.production ]]; then
  echo "Le fichier $app_dir/.env.production est absent"
  exit 1
fi

echo "Récupération du tag $tag"
git fetch --force origin "refs/tags/${tag}:refs/tags/${tag}"
git checkout --detach "$tag"

export BACKEND_IMAGE="ghcr.io/yassine-re/partybox-backend:${tag}"
export FRONTEND_IMAGE="ghcr.io/yassine-re/partybox-frontend:${tag}"

compose=(
  docker compose
  --env-file .env.production
)

echo "Téléchargement des images Docker"
"${compose[@]}" pull

echo "Démarrage de PartyBox"
"${compose[@]}" up -d --no-build --remove-orphans

echo "Attente du healthcheck"

for attempt in {1..24}; do
  if curl --fail --silent http://127.0.0.1:3008/api/health; then
    echo
    echo "Déploiement $tag réussi"
    exit 0
  fi

  echo "Tentative $attempt/24"
  sleep 5
done

echo "Échec du healthcheck"
"${compose[@]}" ps
"${compose[@]}" logs --tail=100 backend frontend
exit 1