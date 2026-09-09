package vision

type Verdict string

const (
	Valid     Verdict = "valid"
	Invalid   Verdict = "invalid"
	Uncertain Verdict = "uncertain"
)

type ValidationRequest struct {
	MissionText string
	Image       []byte
	MediaType   string
}

type ValidationResult struct {
	Verdict    Verdict `json:"verdict"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}
