package md5

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
)

// Digest is the 16-byte MD5 digest of an EVE resfile asset, as stored in manifest indexes.
type Digest [md5.Size]byte

var errInvalid = errors.New("invalid md5 checksum")

// Parse decodes a 32-character hex MD5 from a manifest checksum column.
func Parse(s string) (Digest, error) {
	if s == "" {
		return Digest{}, nil
	}
	if len(s) != md5.Size*2 {
		return Digest{}, fmt.Errorf("%w: expected %d hex characters, got %d", errInvalid, md5.Size*2, len(s))
	}
	var out Digest
	if _, err := hex.Decode(out[:], []byte(s)); err != nil {
		return Digest{}, fmt.Errorf("%w: %w", errInvalid, err)
	}
	return out, nil
}

// IsZero reports whether the digest is unset.
func (h Digest) IsZero() bool {
	return h == Digest{}
}

// String returns the lowercase hex form used in manifest files.
func (h Digest) String() string {
	return hex.EncodeToString(h[:])
}

// Sum returns the digest as the same array type crypto/md5.Sum produces.
func (h Digest) Sum() [md5.Size]byte {
	return [md5.Size]byte(h)
}
