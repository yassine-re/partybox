package deviceauth

import "testing"

func TestDeviceTokenRoundTripAndValidation(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidToken(token, HashToken(token)) {
		t.Fatal("generated token did not match its hash")
	}
	if ValidToken("wrong", HashToken(token)) || ValidToken(token, HashToken(token+"x")) {
		t.Fatal("invalid token accepted")
	}
}
