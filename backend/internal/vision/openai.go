package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIValidator struct {
	apiKey, model, detail, baseURL string
	client                         *http.Client
}

func NewOpenAIValidator(key, model, detail, baseURL string) (*OpenAIValidator, error) {
	key, model, detail = strings.TrimSpace(key), strings.TrimSpace(model), strings.TrimSpace(detail)
	if detail == "" {
		detail = "low"
	}
	if detail != "low" && detail != "high" {
		return nil, errors.New("OPENAI_VISION_DETAIL doit valoir low ou high")
	}
	if key == "" || model == "" {
		return nil, errors.New("vision requires an API key and model")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIValidator{apiKey: key, model: model, detail: detail, baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{
		Timeout:       40 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (v *OpenAIValidator) Validate(ctx context.Context, input ValidationRequest) (ValidationResult, error) {
	var result ValidationResult
	schema := map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"verdict":    map[string]any{"type": "string", "enum": []string{"valid", "invalid", "uncertain"}},
			"confidence": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"reason":     map[string]any{"type": "string", "minLength": 1, "maxLength": MaxReasonLength},
		}, "required": []string{"verdict", "confidence", "reason"},
	}
	body, err := json.Marshal(map[string]any{
		"model": v.model, "store": false, "max_output_tokens": 1000,
		"instructions": systemPrompt,
		"input": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "input_text", "text": "Mission exacte à vérifier :\n" + input.MissionText},
			map[string]any{"type": "input_image", "detail": v.detail, "image_url": "data:" + input.MediaType + ";base64," + base64.StdEncoding.EncodeToString(input.Image)},
		}}},
		"text": map[string]any{"format": map[string]any{"type": "json_schema", "name": "mission_proof", "strict": true, "schema": schema}},
	})
	if err != nil {
		return result, errors.New("vision request encoding failed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return result, errors.New("vision request creation failed")
	}
	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	// Never propagate raw provider bodies, request data, URLs or API keys to logs.
	if err != nil {
		return result, errors.New("vision provider unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("vision provider HTTP %d", resp.StatusCode)
	}
	const maxResponse = 64 << 10
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(raw) > maxResponse {
		return result, errors.New("invalid vision response size")
	}
	var response struct {
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if json.Unmarshal(raw, &response) != nil || response.Status != "completed" {
		return result, errors.New("vision response incomplete")
	}
	var output strings.Builder
	for _, item := range response.Output {
		if item.Type != "message" {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "refusal" {
				return result, errors.New("vision provider declined evaluation")
			}
			if part.Type == "output_text" {
				output.WriteString(part.Text)
			}
		}
	}
	// Pointers reject missing/null required fields (notably confidence=0).
	var decoded struct {
		Verdict    *Verdict `json:"verdict"`
		Confidence *float64 `json:"confidence"`
		Reason     *string  `json:"reason"`
	}
	decoder := json.NewDecoder(strings.NewReader(output.String()))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&decoded) != nil || decoded.Verdict == nil || decoded.Confidence == nil || decoded.Reason == nil {
		return result, errors.New("invalid structured vision response")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return result, errors.New("unexpected vision response content")
	}
	result = ValidationResult{*decoded.Verdict, *decoded.Confidence, *decoded.Reason}
	return result, CheckResult(result)
}
