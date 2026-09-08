package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"partybox/backend/internal/models"
)

const maxOpenAIResponseBytes = 2 << 20

func missionsJSONSchema(count int) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"missions": map[string]any{
				"type":     "array",
				"minItems": count,
				"maxItems": count,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"text": map[string]any{
							"type":        "string",
							"description": "Texte clair et direct de la mission ou de la recherche",
						},
						"category": map[string]any{
							"type":        "string",
							"description": "Catégorie courte de la mission (ex: Conversation, Ambiance, color, object, etc.)",
						},
						"difficulty": map[string]any{
							"type":        "integer",
							"description": "Niveau de difficulté entre 1 et 3",
						},
						"points": map[string]any{
							"type":        "integer",
							"description": "Points attribués (entre 1 et 100)",
						},
					},
					"required":             []string{"text", "category", "difficulty", "points"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"missions"},
		"additionalProperties": false,
	}
}

type OpenAIResponsesGenerator struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIResponsesGenerator(apiKey, model, baseURL string) *OpenAIResponsesGenerator {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIResponsesGenerator{
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 40 * time.Second},
	}
}

func (o *OpenAIResponsesGenerator) Generate(ctx context.Context, input GenerationRequest) ([]GeneratedMission, error) {
	if o.apiKey == "" {
		return nil, fmt.Errorf("%w : clé d’API OpenAI non configurée", models.ErrConflict)
	}
	if o.model == "" {
		return nil, fmt.Errorf("%w : modèle OpenAI non configuré", models.ErrConflict)
	}

	missions, err := o.callResponsesAPI(ctx, buildUserPrompt(input), input.Count)
	if err != nil {
		return nil, err
	}
	if err := ValidateMissions(missions, input.Count); err != nil {
		return nil, fmt.Errorf("validation des missions générées : %w", err)
	}
	return missions, nil
}

func (o *OpenAIResponsesGenerator) callResponsesAPI(ctx context.Context, userMsg string, count int) ([]GeneratedMission, error) {
	maxOutputTokens := count * 120
	if maxOutputTokens < 1000 {
		maxOutputTokens = 1000
	}
	reqBody := map[string]any{
		"model":             o.model,
		"store":             false,
		"max_output_tokens": maxOutputTokens,
		"input": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMsg},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "missions_catalog",
				"strict": true,
				"schema": missionsJSONSchema(count),
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := o.baseURL + "/responses"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur réseau OpenAI : %w", err)
	}
	defer resp.Body.Close()

	rawResp, err := io.ReadAll(io.LimitReader(resp.Body, maxOpenAIResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(rawResp) > maxOpenAIResponseBytes {
		return nil, fmt.Errorf("%w : réponse OpenAI trop volumineuse", models.ErrInvalid)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("erreur OpenAI (HTTP %d) : %s", resp.StatusCode, string(rawResp))
	}

	var respData struct {
		Status     string `json:"status"`
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(rawResp, &respData); err != nil {
		return nil, fmt.Errorf("décodage réponse OpenAI Responses : %w", err)
	}
	if respData.Status != "" && respData.Status != "completed" {
		return nil, fmt.Errorf("%w : génération OpenAI non terminée (statut %s)", models.ErrInvalid, respData.Status)
	}

	jsonContent := respData.OutputText
	if jsonContent == "" {
		var output strings.Builder
		for _, item := range respData.Output {
			for _, part := range item.Content {
				if part.Type == "output_text" && part.Text != "" {
					output.WriteString(part.Text)
				}
			}
		}
		jsonContent = output.String()
	}
	if jsonContent == "" {
		return nil, fmt.Errorf("%w : réponse vide reçue de l’IA", models.ErrInvalid)
	}

	var catalog struct {
		Missions []GeneratedMission `json:"missions"`
	}
	if err := json.Unmarshal([]byte(jsonContent), &catalog); err != nil {
		return nil, fmt.Errorf("%w : format JSON structuré invalide : %v", models.ErrInvalid, err)
	}
	return catalog.Missions, nil
}
