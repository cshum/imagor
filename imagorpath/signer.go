package imagorpath

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"hash"
)

// Signer imagor URL signature signer
type Signer interface {
	Sign(path string) string
}

// NewDefaultSigner default signer using SHA1 with secret
func NewDefaultSigner(secret string) Signer {
	return NewHMACSigner(sha1.New, 0, secret)
}

// NewHMACSigner custom HMAC alg signer with secret and string length based truncate
func NewHMACSigner(alg func() hash.Hash, truncate int, secret string) *hmacSigner {
	return &hmacSigner{
		alg:      alg,
		truncate: truncate,
		secret:   []byte(secret),
	}
}

type hmacSigner struct {
	alg      func() hash.Hash
	truncate int
	secret   []byte
}

func (s *hmacSigner) Sign(path string) string {
	h := hmac.New(s.alg, s.secret)
	h.Write([]byte(path))
	sig := base64.URLEncoding.EncodeToString(h.Sum(nil))
	if s.truncate > 0 && len(sig) > s.truncate {
		return sig[:s.truncate]
	}
	return sig
}
