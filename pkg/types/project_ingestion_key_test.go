package types

import (
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/valuer"
)

func TestProjectIngestionKeySecretLifecycle(t *testing.T) {
	orgID := valuer.GenerateUUID()
	projectID := valuer.GenerateUUID()
	key, secret, err := NewProjectIngestionKey(orgID, projectID, "collector", nil)
	if err != nil { t.Fatal(err) }
	if strings.Contains(key.SecretHash, secret) { t.Fatal("plaintext secret must not be stored") }
	keyID, err := ParseProjectIngestionKey(secret)
	if err != nil { t.Fatal(err) }
	if keyID != key.KeyID { t.Fatalf("key id = %q, want %q", keyID, key.KeyID) }
	if !key.VerifySecret(secret) { t.Fatal("expected generated secret to verify") }
	if key.VerifySecret(secret + "x") { t.Fatal("unexpected secret verification success") }
	if err := key.Validate(time.Now()); err != nil { t.Fatalf("unexpected validation error: %v", err) }
}

func TestProjectIngestionKeyRejectsExpiredAndRevoked(t *testing.T) {
	expired := time.Now().Add(-time.Minute)
	key, _, err := NewProjectIngestionKey(valuer.GenerateUUID(), valuer.GenerateUUID(), "collector", &expired)
	if err != nil { t.Fatal(err) }
	if err := key.Validate(time.Now()); err == nil { t.Fatal("expected expired key to fail") }
	key.ExpiresAt = nil
	now := time.Now()
	key.RevokedAt = &now
	if err := key.Validate(now); err == nil { t.Fatal("expected revoked key to fail") }
}

func TestParseProjectIngestionKeyRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"", "wrong", "snz_proj_bad.secret", "snz_proj_018f0f2a-7b5c-7a00-8000-000000000001.not-base64"} {
		if _, err := ParseProjectIngestionKey(value); err == nil { t.Errorf("expected %q to be rejected", value) }
	}
}
