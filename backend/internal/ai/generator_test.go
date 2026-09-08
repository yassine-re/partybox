package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"partybox/backend/internal/ai"
	"partybox/backend/internal/models"
)

func TestValidateMissions(t *testing.T) {
	validMissions := func(n int) []ai.GeneratedMission {
		res := make([]ai.GeneratedMission, n)
		for i := range res {
			res[i] = ai.GeneratedMission{
				Text:       "Fais dire le mot cactus à un autre joueur.",
				Category:   "Conversation",
				Difficulty: 2,
				Points:     20,
			}
		}
		return res
	}

	t.Run("valid_catalog", func(t *testing.T) {
		err := ai.ValidateMissions(validMissions(20), 20)
		if err != nil {
			t.Fatalf("expected valid catalog, got %v", err)
		}
	})

	t.Run("empty_catalog", func(t *testing.T) {
		err := ai.ValidateMissions(nil, 20)
		if err == nil {
			t.Fatal("expected error on empty catalog")
		}
	})

	t.Run("insufficient_count", func(t *testing.T) {
		err := ai.ValidateMissions(validMissions(2), 20)
		if err == nil {
			t.Fatal("expected error on insufficient count")
		}
	})

	t.Run("excessive_count", func(t *testing.T) {
		err := ai.ValidateMissions(validMissions(21), 20)
		if err == nil {
			t.Fatal("expected error when the catalog exceeds the requested count")
		}
	})

	t.Run("empty_text", func(t *testing.T) {
		missions := validMissions(10)
		missions[0].Text = "   "
		err := ai.ValidateMissions(missions, 10)
		if err == nil {
			t.Fatal("expected error on empty text")
		}
	})

	t.Run("invalid_difficulty", func(t *testing.T) {
		missions := validMissions(10)
		missions[0].Difficulty = 5
		err := ai.ValidateMissions(missions, 10)
		if err == nil {
			t.Fatal("expected error on difficulty > 3")
		}
		missions[0].Difficulty = 0
		err = ai.ValidateMissions(missions, 10)
		if err == nil {
			t.Fatal("expected error on difficulty < 1")
		}
	})

	t.Run("invalid_points", func(t *testing.T) {
		missions := validMissions(10)
		missions[0].Points = 120
		err := ai.ValidateMissions(missions, 10)
		if err == nil {
			t.Fatal("expected error on points > 100")
		}
		missions[0].Points = 0
		err = ai.ValidateMissions(missions, 10)
		if err == nil {
			t.Fatal("expected error on points < 1")
		}
	})
}

func TestOpenAIResponsesGeneratorMockHTTP(t *testing.T) {
	mockCatalog := map[string]any{
		"missions": []map[string]any{
			{
				"text":       "Fais fredonner une chanson à un autre joueur.",
				"category":   "Musique",
				"difficulty": 1,
				"points":     15,
			},
			{
				"text":       "Trouve un objet plus vieux que toi.",
				"category":   "exploration",
				"difficulty": 2,
				"points":     20,
			},
			{
				"text":       "Obtiens un check de trois personnes différentes.",
				"category":   "Interaction",
				"difficulty": 2,
				"points":     20,
			},
			{
				"text":       "Fais dire le mot pingouin.",
				"category":   "Conversation",
				"difficulty": 1,
				"points":     10,
			},
			{
				"text":       "Convaincs deux personnes de faire la même pose.",
				"category":   "Collectif",
				"difficulty": 3,
				"points":     25,
			},
		},
	}
	catalogJSON, _ := json.Marshal(mockCatalog)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/responses" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request: %v", err)
			}
			if body["model"] != "test-model" {
				t.Errorf("unexpected model: %v", body["model"])
			}
			if store, ok := body["store"].(bool); !ok || store {
				t.Errorf("Responses request must set store=false, got %v", body["store"])
			}
			textConfig, _ := body["text"].(map[string]any)
			format, _ := textConfig["format"].(map[string]any)
			schema, _ := format["schema"].(map[string]any)
			properties, _ := schema["properties"].(map[string]any)
			missionArray, _ := properties["missions"].(map[string]any)
			if missionArray["minItems"] != float64(5) || missionArray["maxItems"] != float64(5) {
				t.Errorf("mission count is not bounded in schema: %+v", missionArray)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "resp_123",
				"status": "completed",
				"output": []map[string]any{{
					"type": "message",
					"content": []map[string]any{{
						"type": "output_text",
						"text": string(catalogJSON),
					}},
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	gen := ai.NewOpenAIResponsesGenerator("test-key", "test-model", server.URL)
	missions, err := gen.Generate(context.Background(), ai.GenerationRequest{
		Mode:      models.ModeSecretMissions,
		Vibe:      "fun",
		Intensity: 7,
		Count:     5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missions) != 5 {
		t.Fatalf("expected 5 missions, got %d", len(missions))
	}
	if missions[0].Text != "Fais fredonner une chanson à un autre joueur." {
		t.Fatalf("unexpected mission text: %s", missions[0].Text)
	}
}
