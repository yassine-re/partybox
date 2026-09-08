import type { GameMode } from "$lib/api/types";

// Presentation only: the backend remains authoritative for rules and scores.
export const DEFAULT_GAME_MODE: GameMode = "secret_missions";

interface ModePresentation {
  label: string;
  description: string;
  minPlayers: number;
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
};
