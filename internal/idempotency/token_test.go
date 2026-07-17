package idempotency

import "testing"

func TestGenerateAndHash(t *testing.T) {
	rawToken, err := Generate()
	if err != nil {
		t.Fatalf("Generate returned an error: %v", err)
	}

	if rawToken == "" {
		t.Fatal("Generate returned an empty token")
	}

	firstHash, err := Hash(rawToken)
	if err != nil {
		t.Fatalf("Hash returned an error: %v", err)
	}

	secondHash, err := Hash(rawToken)
	if err != nil {
		t.Fatalf("Hash returned an error: %v", err)
	}

	if firstHash != secondHash {
		t.Fatal("Hash was not deterministic")
	}

	if len(firstHash) != 64 {
		t.Fatalf(
			"expected a 64-character SHA-256 hash, got %d",
			len(firstHash),
		)
	}
}

func TestHashRejectsInvalidTokens(t *testing.T) {
	testCases := []string{
		"",
		"not-base64!",
		"dG9vLXNob3J0",
	}

	for _, testCase := range testCases {
		if _, err := Hash(testCase); err == nil {
			t.Fatalf(
				"expected token %q to be rejected",
				testCase,
			)
		}
	}
}
