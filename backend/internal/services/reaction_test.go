package services

import (
	"testing"
	"time"
)

func TestSoloReactionPoints(t *testing.T) {
	tests := []struct {
		name       string
		reactionMS int
		falseStart bool
		timeout    bool
		want       int
	}{
		{"under 200", 199, false, false, 150},
		{"200 through 299", 200, false, false, 100},
		{"300 through 399", 300, false, false, 75},
		{"400 through 599", 400, false, false, 50},
		{"600 and above", 600, false, false, 25},
		{"false start", 0, true, false, 0},
		{"timeout", 0, false, true, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SoloReactionPoints(test.reactionMS, test.falseStart, test.timeout); got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestParseReactionConfig(t *testing.T) {
	values := map[string]string{
		"REACTION_ENABLED": "true", "REACTION_MIN_INTERVAL_SECONDS": "15",
		"REACTION_MAX_INTERVAL_SECONDS": "30", "DEVICE_ONLINE_TIMEOUT_SECONDS": "8",
	}
	config, err := ParseReactionConfig(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if config.MinInterval != 15*time.Second || config.MaxInterval != 30*time.Second || config.DeviceOnline != 8*time.Second {
		t.Fatalf("unexpected config: %+v", config)
	}
	values["REACTION_MAX_INTERVAL_SECONDS"] = "10"
	if _, err = ParseReactionConfig(func(key string) string { return values[key] }); err == nil {
		t.Fatal("invalid min/max accepted")
	}
}
