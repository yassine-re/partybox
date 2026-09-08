INSERT INTO boxes(id, name) VALUES ('PB001', 'La PartyBox du salon') ON CONFLICT (id) DO NOTHING;

INSERT INTO missions(id, text, points, category, difficulty) VALUES
(1, 'Fais dire le mot « pingouin » à un autre joueur.', 15, 'Conversation', 2),
(2, 'Convaincs quelqu’un de changer la musique.', 10, 'Ambiance', 1),
(3, 'Fais prendre une photo de groupe sans révéler ta mission, avec l’accord de chacun.', 20, 'Collectif', 3),
(4, 'Fais rire deux joueurs différents.', 15, 'Ambiance', 2),
(5, 'Fais en sorte que quelqu’un te propose un verre d’eau.', 10, 'Interaction', 1),
(6, 'Fais prononcer le nom d’un pays à un autre joueur.', 10, 'Conversation', 1),
(7, 'Obtiens un check de trois personnes différentes.', 15, 'Interaction', 2),
(8, 'Amène quelqu’un à raconter son dernier voyage.', 10, 'Conversation', 1),
(9, 'Lance un applaudissement suivi par au moins deux joueurs.', 20, 'Collectif', 3),
(10, 'Fais fredonner une chanson à un autre joueur.', 15, 'Musique', 2),
(11, 'Convaincs deux personnes de faire la même pose pendant cinq secondes.', 20, 'Collectif', 3),
(12, 'Fais deviner ton animal préféré sans dire son nom.', 15, 'Conversation', 2),
(13, 'Fais dire « c’est pas faux » à quelqu’un.', 20, 'Conversation', 3),
(14, 'Trouve un point commun inattendu avec un autre joueur et fais-le annoncer au groupe.', 15, 'Interaction', 2),
(15, 'Fais raconter une blague à un autre joueur.', 10, 'Ambiance', 1),
(16, 'Amène deux joueurs à débattre de leur meilleur film.', 15, 'Conversation', 2),
(17, 'Fais inventer un nom d’équipe à trois joueurs.', 20, 'Collectif', 3),
(18, 'Obtiens une recommandation de chanson de quelqu’un.', 10, 'Musique', 1)
ON CONFLICT (id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('missions', 'id'), (SELECT max(id) FROM missions));
