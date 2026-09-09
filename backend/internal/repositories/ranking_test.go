package repositories

import (
	"math"
	"testing"

	mlranker "partybox/backend/internal/ml"
)

func TestChooseRankedCandidateRestrictsRandomnessToTopFive(t *testing.T) {
	candidates := make([]missionCandidate, 8)
	for index := range candidates {
		candidates[index] = missionCandidate{ID: index + 1, Context: mlranker.MissionContext{Points: index + 1}}
		candidates[index].Score = float64(candidates[index].Context.Points)
	}
	if got := chooseRankedCandidate(candidates, func(limit int) int {
		if limit != mlTopCount {
			t.Fatalf("selection limit=%d, want %d", limit, mlTopCount)
		}
		return limit - 1
	}); got != 4 {
		t.Fatalf("selected candidate=%d, want fifth-best ID 4", got)
	}
}

func TestChooseRankedCandidateFallsBackSafelyOnInvalidScore(t *testing.T) {
	candidates := []missionCandidate{{ID: 7, Score: math.NaN()}, {ID: 8, Score: 1}}
	if got := chooseRankedCandidate(candidates, func(int) int { return 0 }); got != 7 {
		t.Fatalf("invalid scorer should retain unranked fallback, got %d", got)
	}
}
