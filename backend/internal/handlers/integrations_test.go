package handlers

import "testing"

func TestVerifyHMAC(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	secret := "super-secret"
	signature := "bef5c2e5ffdbcbc6a43bf41e12b9fe9d0c1b7827cc3ea0b5efce05bc0b368ff7"
	if !verifyHMAC(payload, signature, secret) {
		t.Fatal("expected valid signature to verify")
	}
	if verifyHMAC(payload, "bad", secret) {
		t.Fatal("expected invalid signature to fail")
	}
}
