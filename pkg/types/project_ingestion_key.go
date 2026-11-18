package types

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/uptrace/bun"
)

var (
	ErrProjectIngestionKeyNotFound = errors.MustNewCode("project_ingestion_key_not_found")
	ErrProjectIngestionKeyInvalid = errors.MustNewCode("project_ingestion_key_invalid")
	ErrProjectIngestionKeyExpired = errors.MustNewCode("project_ingestion_key_expired")
)

const projectIngestionKeyPrefix = "snz_proj_"

// ProjectIngestionKey stores only a hash of the secret. The plaintext secret is
// returned to the caller once by NewProjectIngestionKey and must never be persisted.
type ProjectIngestionKey struct {
	bun.BaseModel `bun:"table:project_ingestion_key,alias:project_ingestion_key"`

	Identifiable
	OrgID       valuer.UUID `bun:"org_id,type:text,notnull" json:"orgId"`
	ProjectID   valuer.UUID `bun:"project_id,type:text,notnull" json:"projectId"`
	Name        string `bun:"name,type:text,notnull" json:"name"`
	KeyID       string `bun:"key_id,type:text,notnull" json:"keyId"`
	SecretHash  string `bun:"secret_hash,type:text,notnull" json:"-"`
	ExpiresAt   *time.Time `bun:"expires_at,nullzero" json:"expiresAt,omitempty"`
	RevokedAt   *time.Time `bun:"revoked_at,nullzero" json:"revokedAt,omitempty"`
	LastUsedAt  *time.Time `bun:"last_used_at,nullzero" json:"lastUsedAt,omitempty"`
	TimeAuditable
}

// NewProjectIngestionKey creates a high-entropy bearer secret. The returned
// secret is the only copy of the plaintext; callers must show it only once.
func NewProjectIngestionKey(orgID, projectID valuer.UUID, name string, expiresAt *time.Time) (*ProjectIngestionKey, string, error) {
	name = strings.TrimSpace(name)
	if orgID.IsZero() || projectID.IsZero() || name == "" || len(name) > 80 {
		return nil, "", errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "organization, project and a name of 1-80 characters are required")
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, "", err
	}
	keyID := valuer.GenerateUUID().String()
	secret := projectIngestionKeyPrefix + keyID + "." + base64.RawURLEncoding.EncodeToString(secretBytes)
	now := time.Now().UTC()
	key := &ProjectIngestionKey{
		Identifiable: Identifiable{ID: valuer.GenerateUUID()},
		OrgID: orgID,
		ProjectID: projectID,
		Name: name,
		KeyID: keyID,
		SecretHash: hashProjectIngestionSecret(secret),
		ExpiresAt: expiresAt,
		TimeAuditable: TimeAuditable{CreatedAt: now, UpdatedAt: now},
	}
	return key, secret, nil
}

func ParseProjectIngestionKey(secret string) (string, error) {
	if !strings.HasPrefix(secret, projectIngestionKeyPrefix) {
		return "", errors.New(errors.TypeUnauthenticated, ErrProjectIngestionKeyInvalid, "invalid project ingestion credential")
	}
	parts := strings.SplitN(strings.TrimPrefix(secret, projectIngestionKeyPrefix), ".", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", errors.New(errors.TypeUnauthenticated, ErrProjectIngestionKeyInvalid, "invalid project ingestion credential")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(decoded) != 32 {
		return "", errors.New(errors.TypeUnauthenticated, ErrProjectIngestionKeyInvalid, "invalid project ingestion credential")
	}
	if _, err := valuer.NewUUID(parts[0]); err != nil {
		return "", errors.New(errors.TypeUnauthenticated, ErrProjectIngestionKeyInvalid, "invalid project ingestion credential")
	}
	return parts[0], nil
}

func (key *ProjectIngestionKey) VerifySecret(secret string) bool {
	if key == nil || key.SecretHash == "" {
		return false
	}
	expected := hashProjectIngestionSecret(secret)
	return subtle.ConstantTimeCompare([]byte(key.SecretHash), []byte(expected)) == 1
}

func (key *ProjectIngestionKey) Validate(now time.Time) error {
	if key == nil || key.RevokedAt != nil {
		return errors.New(errors.TypeUnauthenticated, ErrProjectIngestionKeyInvalid, "project ingestion credential is revoked or invalid")
	}
	if key.ExpiresAt != nil && !now.Before(*key.ExpiresAt) {
		return errors.New(errors.TypeUnauthenticated, ErrProjectIngestionKeyExpired, "project ingestion credential has expired")
	}
	return nil
}

func hashProjectIngestionSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
