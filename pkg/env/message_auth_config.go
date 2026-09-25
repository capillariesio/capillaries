package env

import (
	"github.com/capillariesio/capillaries/pkg/msgsig"
)

// MessageSignConfig holds the producer-side (webapi, toolbelt) message-signing material.
//
// The private key is env-only (json:"-") so it can never be read from a config file. Daemons
// never populate this - only producers sign.
type MessageSignConfig struct {
	ActiveKid  string `json:"active_kid,omitempty" env:"CAPI_MSG_SIGN_ACTIVE_KID, overwrite"`
	PrivateKey string `json:"private_key,omitempty" env:"CAPI_MSG_SIGN_PRIVATE_KEY, overwrite"` // base64 Ed25519 seed or full key; secret
}

// Enabled reports whether signing is configured.
func (c *MessageSignConfig) Enabled() bool {
	return c.ActiveKid != "" && c.PrivateKey != ""
}

// NewSigner returns a signer when signing is configured, or (nil, nil) when it is not (in which
// case producers send unsigned messages - the legacy behavior).
func (c *MessageSignConfig) NewSigner() (*msgsig.Signer, error) {
	if !c.Enabled() {
		return nil, nil
	}
	return msgsig.NewSigner(c.ActiveKid, c.PrivateKey)
}

// MessageVerifyConfig holds the consumer-side (daemon) verification material. Public keys are
// not secret and may live in the config file.
type MessageVerifyConfig struct {
	Mode       string            `json:"mode,omitempty" env:"CAPI_MSG_VERIFY_MODE, overwrite"`               // off | permissive | require
	PublicKeys map[string]string `json:"public_keys,omitempty" env:"CAPI_MSG_VERIFY_PUBLIC_KEYS, overwrite"` // kid -> base64 Ed25519 public key; according to https://github.com/sethvargo/go-envconfig, `export MYVAR="a|b,c|d"` -> map[string]string{"a":"b", "c":"d"}
}

// NewVerifier builds a verifier from config. An empty/off mode yields a non-nil verifier that
// passes messages through unchanged.
func (c *MessageVerifyConfig) NewVerifier() (*msgsig.Verifier, error) {
	mode, err := msgsig.ParseMode(c.Mode)
	if err != nil {
		return nil, err
	}
	return msgsig.NewVerifier(mode, c.PublicKeys)
}
