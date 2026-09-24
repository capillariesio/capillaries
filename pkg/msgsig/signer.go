package msgsig

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Signer seals a serialized message into a SignedEnvelope. It is held only by producers
// (webapi, toolbelt); the daemon never constructs one.
type Signer struct {
	kid  string
	priv ed25519.PrivateKey
}

// NewSigner builds a Signer from an active key id and a base64-encoded Ed25519 private key.
// The key may be either a 32-byte seed or a full 64-byte private key. The key is secret and is
// expected to come from the environment, never a config file.
func NewSigner(kid, privateKeyBase64 string) (*Signer, error) {
	if strings.TrimSpace(kid) == "" {
		return nil, errors.New("message signing: active key id is empty")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(privateKeyBase64))
	if err != nil {
		return nil, fmt.Errorf("message signing: cannot base64-decode private key: %w", err)
	}
	var priv ed25519.PrivateKey
	switch len(raw) {
	case ed25519.SeedSize:
		priv = ed25519.NewKeyFromSeed(raw)
	case ed25519.PrivateKeySize:
		priv = ed25519.PrivateKey(raw)
	default:
		return nil, fmt.Errorf("message signing: private key must be %d (seed) or %d (full) bytes, got %d",
			ed25519.SeedSize, ed25519.PrivateKeySize, len(raw))
	}
	return &Signer{kid: strings.TrimSpace(kid), priv: priv}, nil
}

// Seal wraps payload (the JSON bytes of a wfmodel.Message) in a signed envelope and returns the
// bytes to put on the wire.
func (s *Signer) Seal(payload []byte) ([]byte, error) {
	env := SignedEnvelope{
		Version: envelopeVersion,
		Alg:     algEd25519,
		KeyId:   s.kid,
		Payload: payload,
	}
	env.Signature = ed25519.Sign(s.priv, signedRegion(env.Version, env.Alg, env.KeyId, env.Payload))
	return json.Marshal(env)
}
