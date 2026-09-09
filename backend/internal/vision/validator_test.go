package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageValidation(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var jpg, pngData bytes.Buffer
	if err := jpeg.Encode(&jpg, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	// Synthetic 1x1 WebP produced by a browser canvas (no personal image).
	webpData, err := base64.StdEncoding.DecodeString("UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAUAmJaQAA3AA/v02aAA=")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", jpg.Bytes(), "image/jpeg"}, {"png", pngData.Bytes(), "image/png"},
		{"webp", webpData, "image/webp"},
		{"empty", nil, ""}, {"fake JPEG", []byte("this is not an image"), ""},
		{"truncated PNG", pngData.Bytes()[:40], ""}, {"too large", make([]byte, MaxImageBytes+1), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CheckImage(tc.data)
			if got != tc.want || (err == nil) != (tc.want != "") {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestResultValidation(t *testing.T) {
	for _, v := range []Verdict{Valid, Invalid, Uncertain} {
		if err := CheckResult(ValidationResult{v, .9, "Un objet est visible."}); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []ValidationResult{
		{"unknown", .5, "x"}, {Valid, -.1, "x"}, {Valid, 1.1, "x"}, {Valid, math.NaN(), "x"}, {Valid, math.Inf(1), "x"},
		{Valid, .9, ""}, {Valid, .9, "  "}, {Valid, .9, strings.Repeat("é", MaxReasonLength+1)}, {Valid, .9, "raw\ntext"},
	} {
		if CheckResult(r) == nil {
			t.Fatalf("accepted invalid result: %+v", r)
		}
	}
}

func TestOpenAIContractAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name, status, output string
		httpStatus           int
		wantOK               bool
	}{
		{"valid", "completed", `{"verdict":"valid","confidence":0.92,"reason":"Une tasse rouge est visible."}`, 200, true},
		{"invalid", "completed", `{"verdict":"invalid","confidence":0.8,"reason":"Il manque un objet."}`, 200, true},
		{"uncertain", "completed", `{"verdict":"uncertain","confidence":0,"reason":"Photo sombre."}`, 200, true},
		{"missing confidence", "completed", `{"verdict":"valid","reason":"OK"}`, 200, false},
		{"null confidence", "completed", `{"verdict":"valid","confidence":null,"reason":"OK"}`, 200, false},
		{"out of range", "completed", `{"verdict":"valid","confidence":2,"reason":"OK"}`, 200, false},
		{"extra field", "completed", `{"verdict":"valid","confidence":1,"reason":"OK","image":"secret"}`, 200, false},
		{"trailing JSON", "completed", `{"verdict":"valid","confidence":1,"reason":"OK"}{}`, 200, false},
		{"incomplete", "incomplete", `{"verdict":"valid","confidence":1,"reason":"OK"}`, 200, false},
		{"malformed", "completed", `not JSON`, 200, false},
		{"provider error", "failed", "sensitive-provider-body", 500, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer test-only-key" {
					t.Error("wrong request")
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
					return
				}
				if payload["model"] != "configured-vision-model" || payload["store"] != false {
					t.Error("model/store not configured")
				}
				content := payload["input"].([]any)[0].(map[string]any)["content"].([]any)
				if content[0].(map[string]any)["text"] != "Mission exacte à vérifier :\nTrouve quelque chose de rouge." {
					t.Error("mission changed")
				}
				img := content[1].(map[string]any)
				if img["type"] != "input_image" || img["detail"] != "low" || img["image_url"] != "data:image/png;base64,AQID" {
					t.Error("invalid image content")
				}
				format := payload["text"].(map[string]any)["format"].(map[string]any)
				if format["type"] != "json_schema" || format["strict"] != true {
					t.Error("missing structured output")
				}
				w.WriteHeader(tc.httpStatus)
				if tc.httpStatus != 200 {
					io.WriteString(w, tc.output)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"status": tc.status, "output": []any{map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": tc.output}}}}})
			}))
			defer server.Close()
			v, err := NewOpenAIValidator("test-only-key", "configured-vision-model", "", server.URL)
			if err != nil {
				t.Fatal(err)
			}
			_, err = v.Validate(context.Background(), ValidationRequest{"Trouve quelque chose de rouge.", []byte{1, 2, 3}, "image/png"})
			if (err == nil) != tc.wantOK {
				t.Fatalf("unexpected result: %v", err)
			}
			if err != nil && (strings.Contains(err.Error(), "sensitive-provider-body") || strings.Contains(err.Error(), "test-only-key")) {
				t.Fatal("leaked provider data")
			}
		})
	}
	if _, err := NewOpenAIValidator("key", "model", "auto", ""); err == nil {
		t.Fatal("invalid detail accepted")
	}
	v, err := NewOpenAIValidator("key", "model", "high", "")
	if err != nil || v.detail != "high" {
		t.Fatal("high detail unavailable")
	}
}
