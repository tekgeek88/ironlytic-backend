package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

type ArgonParams struct {
	Memory      uint32 // kibibytes
	Time        uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

var DefaultArgonParams = ArgonParams{
	Memory:      64 * 1024, // 64 MB
	Time:        2,
	Parallelism: 1,
	SaltLen:     16,
	KeyLen:      32,
}

// Encoded format:
// $argon2id$v=19$m=65536,t=2,p=1$<salt_b64>$<hash_b64>
func HashPassword(password string, p ArgonParams) (string, error) {
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Parallelism, p.KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.Memory, p.Time, p.Parallelism, b64Salt, b64Hash,
	)
	return encoded, nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	p, salt, hash, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Parallelism, p.KeyLen)

	if subtle.ConstantTimeCompare(candidate, hash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encoded string) (ArgonParams, []byte, []byte, error) {
	var p ArgonParams

	var variant string
	var version int
	_, err := fmt.Sscanf(encoded, "$%s$v=%d$m=%d,t=%d,p=%d$",
		&variant, &version, &p.Memory, &p.Time, &p.Parallelism,
	)
	if err != nil {
		return p, nil, nil, fmt.Errorf("invalid hash format: %w", err)
	}
	if variant != "argon2id" {
		return p, nil, nil, fmt.Errorf("unsupported variant: %s", variant)
	}
	if version != 19 {
		return p, nil, nil, fmt.Errorf("unsupported version: %d", version)
	}

	// Split for salt/hash
	// expected: $argon2id$v=19$m=...,t=...,p=...$salt$hash
	parts := split(encoded, '$')
	if len(parts) < 6 {
		return p, nil, nil, fmt.Errorf("invalid hash parts")
	}

	saltB64 := parts[4]
	hashB64 := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return p, nil, nil, fmt.Errorf("decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return p, nil, nil, fmt.Errorf("decode hash: %w", err)
	}

	p.SaltLen = uint32(len(salt))
	p.KeyLen = uint32(len(hash))

	return p, salt, hash, nil
}

func split(s string, sep byte) []string {
	out := make([]string, 0, 8)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
