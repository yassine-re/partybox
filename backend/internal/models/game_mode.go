package models

type GameMode string

const (
	ModeSecretMissions GameMode = "secret_missions"
	ModeTreasureHunt   GameMode = "treasure_hunt"
	ModeChaos          GameMode = "chaos"
)

type ModeDefinition struct {
	ID                   GameMode
	MinPlayers           int
	SupportsAIGeneration bool
}

// Keep the supported modes and their shared lifecycle rules in one place.
var gameModes = map[GameMode]ModeDefinition{
	ModeSecretMissions: {ID: ModeSecretMissions, MinPlayers: 2, SupportsAIGeneration: true},
	ModeTreasureHunt:   {ID: ModeTreasureHunt, MinPlayers: 2, SupportsAIGeneration: true},
	ModeChaos:          {ID: ModeChaos, MinPlayers: 2, SupportsAIGeneration: true},
}

func LookupGameMode(mode GameMode) (ModeDefinition, bool) {
	definition, ok := gameModes[mode]
	return definition, ok
}

func SupportsAIGeneration(mode GameMode) bool {
	definition, ok := LookupGameMode(mode)
	return ok && definition.SupportsAIGeneration
}
