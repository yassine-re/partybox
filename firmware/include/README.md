Les headers séparent configuration, hardware, réseau et logique pure. Copier
`secrets.example.h` vers `secrets.h` pour une carte locale ; ce dernier est
ignoré par Git et contient le secret unique du portail de configuration, les
identifiants API et le certificat. Le Wi-Fi du client est saisi sur la box.
Tout le pinout vérifié se renseigne uniquement dans
`board_config.h`. Voir le guide complet dans `firmware/README.md`.
