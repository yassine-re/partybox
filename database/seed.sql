INSERT INTO boxes(id, name) VALUES ('PB001', 'La PartyBox du salon') ON CONFLICT (id) DO NOTHING;

INSERT INTO missions(id, text, points, category, difficulty, mode) VALUES
(1, 'Fais dire le mot « pingouin » à un autre joueur.', 15, 'Conversation', 2, 'secret_missions'),
(2, 'Convaincs quelqu’un de changer la musique.', 10, 'Ambiance', 1, 'secret_missions'),
(3, 'Fais prendre une photo de groupe sans révéler ta mission, avec l’accord de chacun.', 20, 'Collectif', 3, 'secret_missions'),
(4, 'Fais rire deux joueurs différents.', 15, 'Ambiance', 2, 'secret_missions'),
(5, 'Fais en sorte que quelqu’un te propose un verre d’eau.', 10, 'Interaction', 1, 'secret_missions'),
(6, 'Fais prononcer le nom d’un pays à un autre joueur.', 10, 'Conversation', 1, 'secret_missions'),
(7, 'Obtiens un check de trois personnes différentes.', 15, 'Interaction', 2, 'secret_missions'),
(8, 'Amène quelqu’un à raconter son dernier voyage.', 10, 'Conversation', 1, 'secret_missions'),
(9, 'Lance un applaudissement suivi par au moins deux joueurs.', 20, 'Collectif', 3, 'secret_missions'),
(10, 'Fais fredonner une chanson à un autre joueur.', 15, 'Musique', 2, 'secret_missions'),
(11, 'Convaincs deux personnes de faire la même pose pendant cinq secondes.', 20, 'Collectif', 3, 'secret_missions'),
(12, 'Fais deviner ton animal préféré sans dire son nom.', 15, 'Conversation', 2, 'secret_missions'),
(13, 'Fais dire « c’est pas faux » à quelqu’un.', 20, 'Conversation', 3, 'secret_missions'),
(14, 'Trouve un point commun inattendu avec un autre joueur et fais-le annoncer au groupe.', 15, 'Interaction', 2, 'secret_missions'),
(15, 'Fais raconter une blague à un autre joueur.', 10, 'Ambiance', 1, 'secret_missions'),
(16, 'Amène deux joueurs à débattre de leur meilleur film.', 15, 'Conversation', 2, 'secret_missions'),
(17, 'Fais inventer un nom d’équipe à trois joueurs.', 20, 'Collectif', 3, 'secret_missions'),
(18, 'Obtiens une recommandation de chanson de quelqu’un.', 10, 'Musique', 1, 'secret_missions')
ON CONFLICT (id) DO NOTHING;

-- Keep the existing Secret Missions IDs. Treasure Hunt has its own seed IDs.
INSERT INTO missions(id, text, points, category, difficulty, mode) VALUES
(101, 'Trouve quelque chose de rouge qui tient dans une main.', 10, 'color', 1, 'treasure_hunt'),
(102, 'Trouve un objet avec une date écrite dessus.', 10, 'object', 1, 'treasure_hunt'),
(103, 'Trouve quelque chose qui fait du bruit.', 10, 'object', 1, 'treasure_hunt'),
(104, 'Trouve l’objet le plus inutile de la pièce.', 15, 'funny', 2, 'treasure_hunt'),
(105, 'Trouve quelque chose de plus vieux que toi.', 20, 'exploration', 3, 'treasure_hunt'),
(106, 'Trouve trois objets de la même couleur.', 15, 'color', 2, 'treasure_hunt'),
(107, 'Trouve quelque chose qui commence par la lettre P.', 10, 'object', 1, 'treasure_hunt'),
(108, 'Trouve un objet que personne d’autre dans le groupe ne possède.', 20, 'exploration', 3, 'treasure_hunt'),
(109, 'Trouve un objet qui pourrait servir dans une apocalypse et explique pourquoi.', 15, 'creative', 2, 'treasure_hunt'),
(110, 'Trouve l’objet le plus bizarre possible.', 15, 'funny', 2, 'treasure_hunt'),
(111, 'Trouve un objet qui ressemble à un visage.', 15, 'creative', 2, 'treasure_hunt'),
(112, 'Trouve quelque chose de doux et quelque chose de rugueux.', 10, 'exploration', 1, 'treasure_hunt'),
(113, 'Trouve un objet minuscule qui peut raconter une grande histoire.', 20, 'creative', 3, 'treasure_hunt'),
(114, 'Trouve un objet avec au moins quatre couleurs différentes.', 15, 'color', 2, 'treasure_hunt'),
(115, 'Trouve un objet et invente-lui une publicité de dix secondes.', 20, 'funny', 3, 'treasure_hunt')
ON CONFLICT (id) DO NOTHING;

INSERT INTO missions(id, text, points, category, difficulty, mode) VALUES
(201, 'Fais faire un high-five à trois personnes.', 15, 'interaction', 2, 'chaos'),
(202, 'Change de place avec quelqu’un sans expliquer pourquoi.', 10, 'improvisation', 1, 'chaos'),
(203, 'Fais applaudir au moins deux personnes.', 15, 'collective', 2, 'chaos'),
(204, 'Convaincs quelqu’un de dire « on est mal ».', 15, 'conversation', 2, 'chaos'),
(205, 'Fais une pose dramatique avec un autre joueur.', 15, 'creative', 2, 'chaos'),
(206, 'Trouve un objet amusant et garde-le avec toi pendant une mission.', 10, 'object', 1, 'chaos'),
(207, 'Lance un faux débat bienveillant sur un sujet complètement inutile.', 20, 'conversation', 3, 'chaos'),
(208, 'Obtiens une photo avec deux autres joueurs, avec leur accord.', 20, 'collective', 3, 'chaos'),
(209, 'Fais choisir un nombre entre 1 et 10 à quelqu’un.', 10, 'interaction', 1, 'chaos'),
(210, 'Fais dire le prénom d’un autre joueur à deux personnes.', 15, 'conversation', 2, 'chaos'),
(211, 'Invente un mini salut secret avec un autre joueur.', 15, 'creative', 2, 'chaos'),
(212, 'Fais marcher deux personnes au ralenti pendant cinq secondes.', 20, 'collective', 3, 'chaos'),
(213, 'Obtiens une recommandation de film en moins de trente secondes.', 10, 'conversation', 1, 'chaos'),
(214, 'Fais imiter un bruit d’animal à quelqu’un.', 15, 'improvisation', 2, 'chaos'),
(215, 'Fais former un cercle à trois personnes pour une raison inventée.', 20, 'collective', 3, 'chaos')
ON CONFLICT (id) DO NOTHING;

SELECT setval(pg_get_serial_sequence('missions', 'id'), (SELECT max(id) FROM missions));
