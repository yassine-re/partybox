package ai

import (
	"fmt"
	"strings"

	"partybox/backend/internal/models"
)

const systemPrompt = `Tu es le Maître du Jeu de PartyBox, une console de jeux de soirée IRL entre amis.
Ton rôle est de concevoir un catalogue de missions immersives, amusantes et adaptées au contexte de la soirée.

RÈGLES ABSOLUES DE SÉCURITÉ ET DE RESPECT :
1. AMUSANT ET BIENVEILLANT : aucune humiliation, aucune moquerie méchante, aucun harcèlement.
2. SÉCURITÉ PHYSIQUE : aucun geste dangereux, aucune cascade, aucun contact physique non consenti, aucune destruction ou dégradation de biens.
3. AUCUNE ACTIVITÉ ILLÉGALE ni contenu sexuel ou explicite.
4. AUCUNE PRESSION à consommer de l'alcool, des drogues ou des substances.
5. RESPECT DE LA VIE PRIVÉE : aucune divulgation de secrets personnels ou intimes, pas de fouille de téléphones ni d'objets intimes.
6. LIBERTÉ DE REFUS : chaque participant doit pouvoir refuser une action sans pénalité sociale.

LANGUE : Génère toutes les missions en français courant, percutant et naturel.`

func buildUserPrompt(req GenerationRequest) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Génère exactement %d missions personnalisées pour le mode de jeu : %s.\n\n", req.Count, req.Mode)
	fmt.Fprintf(&sb, "PARAMÈTRES DE LA SOIRÉE :\n")
	fmt.Fprintf(&sb, "- Ambiance (Vibe) : %s\n", req.Vibe)
	fmt.Fprintf(&sb, "- Intensité (1=très tranquille, 10=survolté) : %d/10\n", req.Intensity)
	if req.PlayerCount > 0 {
		fmt.Fprintf(&sb, "- Nombre de joueurs présents : %d\n", req.PlayerCount)
	}
	if trimmedCtx := strings.TrimSpace(req.Context); trimmedCtx != "" {
		fmt.Fprintf(&sb, "- Contexte spécifique fourni par l’hôte : « %s »\n", trimmedCtx)
	}

	sb.WriteString("\nDIRECTIVES DE GAMEPLAY SELON LE MODE :\n")
	switch req.Mode {
	case models.ModeSecretMissions:
		sb.WriteString(`- Mode : SECRET MISSIONS.
- Les joueurs reçoivent chacun une mission secrète qu'ils doivent accomplir discrètement auprès des autres sans se faire démasquer.
- Exemples de missions : faire dire un mot particulier (« cactus », « pingouin »), lancer un débat absurde, obtenir un check, faire faire une action collective sans l'annoncer.
- Les missions doivent être sociales, subtiles, courtes et impliquer les autres convives.
- Difficulté 1 (Facile, 10-15 pts) : interactions simples et naturelles.
- Difficulté 2 (Intermédiaire, 15-20 pts) : nécessite un peu d'ingéniosité ou d'audace discrète.
- Difficulté 3 (Corsée, 20-30 pts) : challenge collectif ou mot complexe à glisser sans attirer l'attention.
- Catégories suggérées : Conversation, Ambiance, Défi, Collectif, Interaction.
`)
	case models.ModeTreasureHunt:
		sb.WriteString(`- Mode : TREASURE HUNT (Chasse au trésor IRL).
- Les joueurs reçoivent un objectif d'observation ou de découverte d'objet ou de situation dans leur environnement immédiat.
- Exemples de recherches : « Trouve quelque chose de rouge qui tient dans une main », « Trouve l’objet le plus vieux de la pièce », « Trouve quelque chose qui fait du bruit ».
- Tout doit être accessible dans le lieu de la fête (appartement, maison, jardin autorisé), sans jamais pénétrer dans des zones interdites ni fouiller dans des affaires privées.
- Difficulté 1 (Facile, 10-15 pts) : objets ou couleurs très communs.
- Difficulté 2 (Intermédiaire, 15-20 pts) : caractéristiques insolites ou combinaisons d'objets.
- Difficulté 3 (Corsée, 20-30 pts) : objets rares, créatifs ou histoires à inventer sur place.
- Catégories suggérées : color, object, funny, creative, exploration.
`)
	}

	sb.WriteString(`
Assure-toi que les points attribués sont proportionnels à la difficulté (entre 10 et 30 généralement, maximum 100).
Varie les catégories et les difficultés pour donner un catalogue riche et équilibré.`)

	return sb.String()
}
