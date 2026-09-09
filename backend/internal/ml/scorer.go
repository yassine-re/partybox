package ml

import "math"

type MissionContext struct {
	Mode       string
	Category   string
	Difficulty int
	Points     int
	Source     string
	GameSize   int
}

type Scorer interface {
	Score(MissionContext) float64
}

func (m *Model) Score(input MissionContext) float64 {
	value := m.data.Intercept
	value += m.data.CategoricalFeatures["mode"][input.Mode]
	value += m.data.CategoricalFeatures["category"][input.Category]
	value += m.data.CategoricalFeatures["source"][input.Source]
	for feature, raw := range map[string]float64{
		"difficulty": float64(input.Difficulty),
		"points":     float64(input.Points),
		"game_size":  float64(input.GameSize),
	} {
		settings := m.data.NumericFeatures[feature]
		value += settings.Coefficient * ((raw - settings.Mean) / settings.Scale)
	}
	if value >= 0 {
		return 1 / (1 + math.Exp(-value))
	}
	exponential := math.Exp(value)
	return exponential / (1 + exponential)
}
