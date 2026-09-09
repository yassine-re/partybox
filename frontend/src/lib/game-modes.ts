import type { GameMode } from "$lib/api/types";

// Presentation only: the backend remains authoritative for rules and scores.
export const DEFAULT_GAME_MODE: GameMode = "secret_missions";

interface ModePresentation {
  label: string;
  description: string;
  minPlayers: number;
  supportsAIGeneration: boolean;
  symbol: string;
  number: string;
  heading: readonly [string, string, string];
  punchline: string;
  intro: string;
  lobbyDescription: string;
  startHint: string;
  tabLabel: string;
  cardTitle: string;
  canHide: boolean;
  actionLabel: string;
  completionMessage: (points: number) => string;
  alreadyCompletedMessage: string;
  loadingText: string;
  playerLabel: string;
  statsLabel: string;
  completedSingular: string;
  completedPlural: string;
  finalEyebrow: string;
  cardNote: string;
}

export const GAME_MODES = {
  secret_missions: {
    label: "Secret Missions",
    description: "Accomplis discrètement des missions impliquant tes potes sans te faire griller.",
    minPlayers: 2,
    supportsAIGeneration: true,
    symbol: "✳",
    number: "01",
    heading: ["Ce soir,", "tout le monde", "a un"],
    punchline: "secret.",
    intro: "Des missions discrètes. Des amis complices. Une soirée qui ne ressemble à aucune autre.",
    lobbyDescription: "Tout le monde est là ? Une fois la partie lancée, chacun reçoit sa mission. Garde-la pour toi.",
    startHint: "Ta mission sera attribuée au lancement.",
    tabLabel: "Ma mission",
    cardTitle: "MA MISSION SECRÈTE",
    canHide: true,
    actionLabel: "Mission accomplie",
    completionMessage: (points: number) => `Mission accomplie ! +${points} points. Une nouvelle mission t’attend.`,
    alreadyCompletedMessage: "Cette mission était déjà validée. À toi la suivante !",
    loadingText: "Ta mission arrive…",
    playerLabel: "AGENT",
    statsLabel: "MISSION(S)",
    completedSingular: "mission accomplie",
    completedPlural: "missions accomplies",
    finalEyebrow: "LES SECRETS SONT DÉVOILÉS",
    cardNote: "Joue le jeu. Valide seulement quand c’est fait.",
  },
  treasure_hunt: {
    label: "Treasure Hunt",
    description: "Trouve des objets autour de toi, valide tes découvertes et accumule des points.",
    minPlayers: 2,
    supportsAIGeneration: true,
    symbol: "⌖",
    number: "02",
    heading: ["Ce soir,", "le trésor est", "sous tes"],
    punchline: "yeux.",
    intro: "Un regard neuf sur la pièce. Des trouvailles improbables. La chasse est ouverte.",
    lobbyDescription: "Tout le monde est là ? Chacun reçoit un objet à trouver autour de soi. Observe, cherche et montre ta découverte au groupe !",
    startHint: "Ta première recherche sera attribuée au lancement.",
    tabLabel: "Ma recherche",
    cardTitle: "MA RECHERCHE",
    canHide: false,
    actionLabel: "J’ai trouvé",
    completionMessage: (points: number) => `Découverte validée ! +${points} points. Une nouvelle recherche t’attend.`,
    alreadyCompletedMessage: "Cette découverte était déjà validée. À toi la suivante !",
    loadingText: "Ta recherche arrive…",
    playerLabel: "EXPLORATEUR",
    statsLabel: "DÉCOUVERTE(S)",
    completedSingular: "découverte validée",
    completedPlural: "découvertes validées",
    finalEyebrow: "LA CHASSE EST TERMINÉE",
    cardNote: "Valide quand tu as trouvé. (La validation photo arrivera dans une prochaine version).",
  },
  chaos: {
    label: "Chaos",
    description: "Accomplis tes missions pendant que la PartyBox change les règles en plein milieu de la partie.",
    minPlayers: 2,
    supportsAIGeneration: true,
    symbol: "⚡",
    number: "03",
    heading: ["Ce soir,", "les règles", "vont devenir"],
    punchline: "instables.",
    intro: "Des missions imprévisibles. Des règles qui basculent. Personne ne sait ce que la PartyBox prépare.",
    lobbyDescription: "Ici, rien ne reste stable très longtemps. Termine tes missions et prépare-toi aux événements Chaos.",
    startHint: "Ta première mission Chaos sera attribuée au lancement.",
    tabLabel: "Ma mission",
    cardTitle: "MA MISSION CHAOS",
    canHide: false,
    actionLabel: "Défi relevé",
    completionMessage: (points: number) => `Défi relevé ! +${points} points. Le Chaos continue.`,
    alreadyCompletedMessage: "Cette mission était déjà validée. Aucun effet Chaos supplémentaire.",
    loadingText: "Le Chaos choisit ta mission…",
    playerLabel: "AGENT DU CHAOS",
    statsLabel: "DÉFI(S)",
    completedSingular: "défi relevé",
    completedPlural: "défis relevés",
    finalEyebrow: "LE CHAOS RETOMBE",
    cardNote: "Le score et les effets sont toujours calculés par la PartyBox.",
  },
} satisfies Record<GameMode, ModePresentation>;

export const GAME_MODE_OPTIONS = Object.entries(GAME_MODES).map(
  ([id, definition]) => ({ id: id as GameMode, ...definition }),
);

export const MISSION_CATEGORIES: Record<string, string> = {
  color: "Couleur",
  object: "Objet",
  funny: "Insolite",
  creative: "Créativité",
  exploration: "Exploration",
  interaction: "Interaction",
  improvisation: "Improvisation",
  collective: "Collectif",
  conversation: "Conversation",
};
