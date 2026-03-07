package auth

import (
	"github.com/apokalyptik/phpass"
	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptCost is the cost parameter for bcrypt hashing.
	bcryptCost = 12
)

// HashPassword creates a bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks a password against a bcrypt hash.
// Returns true if the password matches.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// VerifyLegacyPassword checks a password against a phpass hash
// (from the WordPress migration). Returns true if the password matches.
func VerifyLegacyPassword(hash, password string) bool {
	p := phpass.New(nil)
	return p.Check([]byte(password), []byte(hash))
}

// CheckPassword verifies a password against bcrypt (primary) and
// phpass (legacy fallback). Returns:
//   - matched: whether the password is correct
//   - needsRehash: whether the old_password matched (caller should rehash to bcrypt)
func CheckPassword(bcryptHash, phpassHash, password string) (matched bool, needsRehash bool) {
	// Try bcrypt first
	if bcryptHash != "" && VerifyPassword(bcryptHash, password) {
		return true, false
	}

	// Try phpass legacy hash
	if phpassHash != "" && VerifyLegacyPassword(phpassHash, password) {
		return true, true // caller should update to bcrypt
	}

	return false, false
}
