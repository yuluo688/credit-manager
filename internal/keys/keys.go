package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/yuluo688/credit-manager/internal/config"
)

const (
	// FormatVersion is the public key format identifier.
	FormatVersion = "v1"
	// Prefix is the full key prefix: tk-v1-<kid>-<secret>
	Prefix = "tk-" + FormatVersion + "-"
	// PrincipalPrefix is a stable non-secret principal for host caller_scope.
	PrincipalPrefix = "cmk:"
	kidBytes        = 10
	secretBytes     = 32
)

var (
	ErrInvalidKey    = errors.New("invalid plugin key")
	ErrUnknownPepper = errors.New("unknown pepper id")
)

// Material is the secret-bearing representation returned only at mint/rotate.
type Material struct {
	// Plaintext is shown once to the administrator.
	Plaintext string
	// Kid is the public key identifier embedded in the plaintext.
	Kid string
	// PepperID identifies which external pepper produced KeyHash.
	PepperID string
	// KeyHash is HMAC-SHA256(pepper, plaintext).
	KeyHash []byte
	// Fingerprint is a non-secret display hash of the kid (never the secret).
	Fingerprint string
	// Principal is the stable host identity for this credential.
	Principal string
	// CallerScope is the host-derived SHA-256 namespace of Principal
	// (sdk/cliproxy/session.CallerScope).
	CallerScope string
}

// Parse extracts the public kid from a Bearer token without verifying the secret.
func Parse(plaintext string) (kid string, err error) {
	plaintext = strings.TrimSpace(plaintext)
	if !strings.HasPrefix(plaintext, Prefix) {
		return "", ErrInvalidKey
	}
	rest := strings.TrimPrefix(plaintext, Prefix)
	kid, secret, ok := strings.Cut(rest, "-")
	if !ok || kid == "" || secret == "" {
		return "", ErrInvalidKey
	}
	if !isBase32NoPad(kid) || !isBase32NoPad(secret) {
		return "", ErrInvalidKey
	}
	return kid, nil
}

// MaterialFromPlaintext validates administrator-provided key material and derives
// its stored representation using the active pepper.
func MaterialFromPlaintext(plaintext string, peppers config.PepperSet) (Material, error) {
	plaintext = strings.TrimSpace(plaintext)
	kid, err := Parse(plaintext)
	if err != nil {
		return Material{}, err
	}
	if peppers.ActiveID == "" || len(peppers.Values[peppers.ActiveID]) == 0 {
		return Material{}, fmt.Errorf("%w: active pepper missing", ErrUnknownPepper)
	}
	hash, err := Hash(plaintext, peppers.ActiveID, peppers)
	if err != nil {
		return Material{}, err
	}
	principal := Principal(kid)
	return Material{
		Plaintext:   plaintext,
		Kid:         kid,
		PepperID:    peppers.ActiveID,
		KeyHash:     hash,
		Fingerprint: Fingerprint(kid),
		Principal:   principal,
		CallerScope: CallerScope(principal),
	}, nil
}

// Mint creates a new plugin key under the active pepper.
func Mint(peppers config.PepperSet) (Material, error) {
	if peppers.ActiveID == "" || len(peppers.Values[peppers.ActiveID]) == 0 {
		return Material{}, fmt.Errorf("%w: active pepper missing", ErrUnknownPepper)
	}
	kidRaw := make([]byte, kidBytes)
	secretRaw := make([]byte, secretBytes)
	if _, err := rand.Read(kidRaw); err != nil {
		return Material{}, err
	}
	if _, err := rand.Read(secretRaw); err != nil {
		return Material{}, err
	}
	kid := encodeBase32(kidRaw)
	secret := encodeBase32(secretRaw)
	plaintext := Prefix + kid + "-" + secret
	hash, err := Hash(plaintext, peppers.ActiveID, peppers)
	if err != nil {
		return Material{}, err
	}
	principal := Principal(kid)
	return Material{
		Plaintext:   plaintext,
		Kid:         kid,
		PepperID:    peppers.ActiveID,
		KeyHash:     hash,
		Fingerprint: Fingerprint(kid),
		Principal:   principal,
		CallerScope: CallerScope(principal),
	}, nil
}

// CallerScope mirrors CLIProxyAPI sdk/cliproxy/session.CallerScope so the
// plugin can resolve identity from host execution metadata.
func CallerScope(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("cli-proxy-api:caller-scope:v1\x00" + value))
	return hex.EncodeToString(sum[:])
}

// Hash computes the verification digest for a plaintext key.
func Hash(plaintext, pepperID string, peppers config.PepperSet) ([]byte, error) {
	pepper, ok := peppers.Values[pepperID]
	if !ok || len(pepper) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrUnknownPepper, pepperID)
	}
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(plaintext))
	return mac.Sum(nil), nil
}

// EncryptPlaintext creates a versioned AES-GCM ciphertext using the active
// key pepper. The stored value is only decryptable while that pepper remains configured.
func EncryptPlaintext(plaintext string, peppers config.PepperSet) ([]byte, error) {
	if peppers.ActiveID == "" || len(peppers.Values[peppers.ActiveID]) == 0 {
		return nil, fmt.Errorf("%w: active pepper missing", ErrUnknownPepper)
	}
	key := encryptionKey(peppers.Values[peppers.ActiveID])
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	value := "v1:" + peppers.ActiveID + ":" + base64.RawStdEncoding.EncodeToString(nonce) + ":" + base64.RawStdEncoding.EncodeToString(sealed)
	return []byte(value), nil
}

// DecryptPlaintext decrypts a value produced by EncryptPlaintext.
func DecryptPlaintext(ciphertext []byte, peppers config.PepperSet) (string, error) {
	parts := strings.Split(string(ciphertext), ":")
	if len(parts) != 4 || parts[0] != "v1" || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("invalid encrypted plugin key")
	}
	pepper, ok := peppers.Values[parts[1]]
	if !ok || len(pepper) == 0 {
		return "", fmt.Errorf("%w: %s", ErrUnknownPepper, parts[1])
	}
	nonce, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.New("invalid encrypted plugin key")
	}
	sealed, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return "", errors.New("invalid encrypted plugin key")
	}
	key := encryptionKey(pepper)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", errors.New("decrypt plugin key")
	}
	return string(plaintext), nil
}

func encryptionKey(pepper []byte) [32]byte {
	return sha256.Sum256(append([]byte("token-quota:key-encryption:v1\x00"), pepper...))
}

// Verify checks plaintext against a stored hash with constant-time comparison.
// It tries the stored pepper first, then remaining peppers for rotation windows.
func Verify(plaintext string, storedHash []byte, storedPepperID string, peppers config.PepperSet) (matchedPepperID string, ok bool) {
	if len(storedHash) == 0 {
		return "", false
	}
	try := func(pepperID string) bool {
		digest, err := Hash(plaintext, pepperID, peppers)
		if err != nil {
			return false
		}
		return subtle.ConstantTimeCompare(digest, storedHash) == 1
	}
	if storedPepperID != "" && try(storedPepperID) {
		return storedPepperID, true
	}
	for id := range peppers.Values {
		if id == storedPepperID {
			continue
		}
		if try(id) {
			return id, true
		}
	}
	return "", false
}

// Principal returns the stable non-secret identity bound to a kid.
func Principal(kid string) string {
	return PrincipalPrefix + kid
}

// KidFromPrincipal extracts the kid from a principal string.
func KidFromPrincipal(principal string) (string, error) {
	principal = strings.TrimSpace(principal)
	if !strings.HasPrefix(principal, PrincipalPrefix) {
		return "", ErrInvalidKey
	}
	kid := strings.TrimPrefix(principal, PrincipalPrefix)
	if kid == "" || !isBase32NoPad(kid) {
		return "", ErrInvalidKey
	}
	return kid, nil
}

// Fingerprint is a short non-secret display value derived only from the public kid.
func Fingerprint(kid string) string {
	sum := sha256.Sum256([]byte("fingerprint:" + kid))
	return hex.EncodeToString(sum[:8])
}

func encodeBase32(raw []byte) string {
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
}

func isBase32NoPad(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '2' && r <= '7':
		case r >= 'A' && r <= 'Z':
		default:
			return false
		}
	}
	return true
}
