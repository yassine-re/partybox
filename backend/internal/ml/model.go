package ml

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"
)

const maxModelBytes = 1 << 20

var expectedFeatures = []string{"mode", "category", "difficulty", "points", "source", "game_size"}

type NumericFeature struct {
	Coefficient float64 `json:"coefficient"`
	Mean        float64 `json:"mean"`
	Scale       float64 `json:"scale"`
}

type portableMetadata struct {
	TrainedAt     string `json:"trained_at"`
	Rows          int    `json:"rows"`
	Games         int    `json:"games"`
	Positive      int    `json:"positive"`
	Negative      int    `json:"negative"`
	BeatsBaseline bool   `json:"beats_baseline"`
}

type portableModel struct {
	SchemaVersion       int                           `json:"schema_version"`
	ModelType           string                        `json:"model_type"`
	FeatureOrder        []string                      `json:"feature_order"`
	Intercept           float64                       `json:"intercept"`
	CategoricalFeatures map[string]map[string]float64 `json:"categorical_features"`
	NumericFeatures     map[string]NumericFeature     `json:"numeric_features"`
	Metadata            portableMetadata              `json:"metadata"`
}

type Model struct{ data portableModel }

func Load(path string) (*Model, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > maxModelBytes {
		return nil, errors.New("ML ranker exceeds 1 MiB")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxModelBytes+1))
	decoder.DisallowUnknownFields()
	var data portableModel
	if err = decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("decode ML ranker: %w", err)
	}
	if err = ensureJSONEnd(decoder); err != nil {
		return nil, err
	}
	if err = validate(data); err != nil {
		return nil, err
	}
	return &Model{data: data}, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("ML ranker contains trailing JSON")
	}
	return fmt.Errorf("decode ML ranker trailer: %w", err)
}

func validate(data portableModel) error {
	if data.SchemaVersion != 1 || data.ModelType != "logistic_regression" {
		return errors.New("unsupported ML ranker format")
	}
	if !slices.Equal(data.FeatureOrder, expectedFeatures) {
		return errors.New("unexpected ML ranker feature order")
	}
	if !data.Metadata.BeatsBaseline {
		return errors.New("ML ranker did not beat its evaluation baseline")
	}
	if strings.TrimSpace(data.Metadata.TrainedAt) == "" || data.Metadata.Rows <= 0 || data.Metadata.Games <= 0 ||
		data.Metadata.Positive < 0 || data.Metadata.Negative < 0 || data.Metadata.Positive+data.Metadata.Negative != data.Metadata.Rows {
		return errors.New("invalid ML ranker metadata")
	}
	if !finite(data.Intercept) {
		return errors.New("invalid ML ranker intercept")
	}
	for _, feature := range []string{"mode", "category", "source"} {
		categories, ok := data.CategoricalFeatures[feature]
		if !ok || len(categories) == 0 {
			return fmt.Errorf("missing categorical feature %s", feature)
		}
		for category, coefficient := range categories {
			if strings.TrimSpace(category) == "" || !finite(coefficient) {
				return fmt.Errorf("invalid category coefficient for %s", feature)
			}
		}
	}
	if len(data.CategoricalFeatures) != 3 {
		return errors.New("unexpected categorical ML features")
	}
	for _, feature := range []string{"difficulty", "points", "game_size"} {
		numeric, ok := data.NumericFeatures[feature]
		if !ok || !finite(numeric.Coefficient) || !finite(numeric.Mean) || !finite(numeric.Scale) || numeric.Scale <= 0 {
			return fmt.Errorf("invalid numeric feature %s", feature)
		}
	}
	if len(data.NumericFeatures) != 3 {
		return errors.New("unexpected numeric ML features")
	}
	return nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
