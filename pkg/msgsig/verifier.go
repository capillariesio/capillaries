package msgsig

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Mode controls how strictly a Verifier treats unsigned messages.
type Mode int

const (
	// ModeOff disables verification: Open returns messages unchanged. Legacy behavior.
	ModeOff Mode = iota
	// ModePermissive verifies a signature when one is present (rejecting bad ones) but accepts
	// unsigned messages, counting them. Used during rollout while producers start signing.
	ModePermissive
	// ModeRequire rejects any message that is not validly signed.
	ModeRequire
)

// ParseMode maps a config string to a Mode. Empty is treated as off.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off":
		return ModeOff, nil
	case "permissive":
		return ModePermissive, nil
	case "require":
		return ModeRequire, nil
	default:
		return ModeOff, fmt.Errorf("message verify: unknown mode %q (expected off, permissive, or require)", s)
	}
}

// Verifier opens signed envelopes and returns the wrapped message bytes. It is held by
// consumers (daemon) and carries only public keys.
type Verifier struct {
	mode Mode
	ring map[string]ed25519.PublicKey
}

// NewVerifier builds a Verifier from a mode and a ring of base64-encoded 32-byte Ed25519 public
// keys, keyed by key id. Public keys are not secret. It is an error to require verification with
// an empty ring (nothing could ever be accepted).
func NewVerifier(mode Mode, publicKeysBase64 map[string]string) (*Verifier, error) {
	ring := make(map[string]ed25519.PublicKey, len(publicKeysBase64))
	for kid, b64 := range publicKeysBase64 {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
		if err != nil {
			return nil, fmt.Errorf("message verify: cannot base64-decode public key %q: %w", kid, err)
		}
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("message verify: public key %q must be %d bytes, got %d",
				kid, ed25519.PublicKeySize, len(raw))
		}
		ring[kid] = ed25519.PublicKey(raw)
	}
	if mode == ModeRequire && len(ring) == 0 {
		return nil, errors.New("message verify: mode is 'require' but no public keys are configured")
	}
	return &Verifier{mode: mode, ring: ring}, nil
}

// Mode reports the verifier's configured mode.
func (v *Verifier) Mode() Mode {
	if v == nil {
		return ModeOff
	}
	return v.mode
}

// Open verifies raw and returns the wrapped wfmodel.Message JSON bytes.
//
// A nil verifier or one in ModeOff passes raw through unchanged. A present but invalid signature
// is always rejected, even in permissive mode - permissive only tolerates the total absence of a
// signature (a legacy message), which it accepts and counts.
func (v *Verifier) Open(raw []byte) ([]byte, error) {
	if v == nil || v.mode == ModeOff {
		return raw, nil
	}

	var env SignedEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Version == 0 {
		// Not a signed envelope: a legacy/unsigned message (no "v" field -> version 0), or not
		// even JSON. Tolerate only in permissive mode.
		if v.mode == ModePermissive {
			VerifyUnsignedCounter.Inc()
			return raw, nil
		}
		VerifyFailCounter.Inc()
		return nil, ErrUnsigned
	}
	if env.Version != envelopeVersion {
		VerifyFailCounter.Inc()
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedVersion, env.Version)
	}
	if env.Alg != algEd25519 {
		VerifyFailCounter.Inc()
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedAlg, env.Alg)
	}
	pub, ok := v.ring[env.KeyId]
	if !ok {
		VerifyFailCounter.Inc()
		return nil, fmt.Errorf("%w: %q", ErrUnknownKey, env.KeyId)
	}
	if !ed25519.Verify(pub, signedRegion(env.Version, env.Alg, env.KeyId, env.Payload), env.Signature) {
		VerifyFailCounter.Inc()
		return nil, ErrBadSignature
	}
	return env.Payload, nil
}
