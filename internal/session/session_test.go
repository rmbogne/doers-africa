package session

import "testing"

func TestNewToken(t *testing.T) {
	rawToken, tokenHash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken returned an error: %v", err)
	}

	if rawToken == "" {
		t.Fatal("raw session token must not be empty")
	}

	if tokenHash == "" {
		t.Fatal("session token hash must not be empty")
	}

	if rawToken == tokenHash {
		t.Fatal("raw token and stored hash must differ")
	}

	if Hash(rawToken) != tokenHash {
		t.Fatal("stored hash does not match raw token")
	}
}

func TestNewTokenGeneratesUniqueValues(t *testing.T) {
	firstToken, _, err := NewToken()
	if err != nil {
		t.Fatalf("first NewToken call failed: %v", err)
	}

	secondToken, _, err := NewToken()
	if err != nil {
		t.Fatalf("second NewToken call failed: %v", err)
	}

	if firstToken == secondToken {
		t.Fatal("session tokens must be unique")
	}
}
