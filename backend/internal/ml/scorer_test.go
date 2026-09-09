package ml

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func fixturePath(name string) string {
	return filepath.Join("..", "..", "..", "ml", "tests", "fixtures", name)
}

func TestPortableModelMatchesPythonScores(t *testing.T) {
	model, err := Load(fixturePath("portable_model.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fixturePath("scoring_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Input struct {
			Mode       string `json:"mode"`
			Category   string `json:"category"`
			Difficulty int    `json:"difficulty"`
			Points     int    `json:"points"`
			Source     string `json:"source"`
			GameSize   int    `json:"game_size"`
		} `json:"input"`
		PythonScore float64 `json:"python_score"`
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		got := model.Score(MissionContext{
			Mode: test.Input.Mode, Category: test.Input.Category,
			Difficulty: test.Input.Difficulty, Points: test.Input.Points,
			Source: test.Input.Source, GameSize: test.Input.GameSize,
		})
		if math.Abs(got-test.PythonScore) >= 1e-6 {
			t.Fatalf("Go score %.10f differs from Python %.10f", got, test.PythonScore)
		}
	}
}

func TestLoadRejectsModelThatDidNotBeatBaseline(t *testing.T) {
	data, err := os.ReadFile(fixturePath("portable_model.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	document["metadata"].(map[string]any)["beats_baseline"] = false
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ranker.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(path); err == nil {
		t.Fatal("expected an unhelpful model to be rejected")
	}
}
