// Package msgsig provides asymmetric (Ed25519) authentication for queue messages.
//
// A queue message body is the trigger that makes a daemon fetch and execute a script URL
// against a keyspace, so anyone who can enqueue a message can drive the framework. To make
// message authenticity verifiable, producers (webapi, toolbelt) wrap the serialized
// wfmodel.Message in a SignedEnvelope and sign it with a private key; consumers (daemon)
// verify the signature against a ring of public keys before acting on the message.
//
// Asymmetric keys are used deliberately: daemons only ever verify, so they hold public keys
// only. A compromised daemon can read the queue but cannot forge a message that other
// daemons will accept. Public keys are not secret and may live in a config file; the
// producer's private key is loaded from the environment only.
package msgsig

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// envelopeVersion is the current wire-format version. A legacy (unsigned) wfmodel.Message
	// has no "v" field and therefore decodes to version 0, which is how Open tells the two apart.
	envelopeVersion = 1
	// algEd25519 is the only signature algorithm accepted. Anything else (including "none") is
	// rejected, so an attacker cannot downgrade a signed message to an unauthenticated one.
	algEd25519 = "Ed25519"
)

// SignedEnvelope wraps the exact JSON bytes of a wfmodel.Message together with the metadata
// needed to verify them. The signature covers the version, algorithm, key id, and payload, so
// none of those can be tampered with (e.g. downgrading Alg or swapping KeyId) without detection.
type SignedEnvelope struct {
	Version   int    `json:"v"`       // wire-format version; 1
	Alg       string `json:"alg"`     // signature algorithm; "Ed25519"
	KeyId     string `json:"kid"`     // id of the key pair that signed this message (enables rotation)
	Payload   []byte `json:"payload"` // exact JSON bytes of the wrapped wfmodel.Message
	Signature []byte `json:"sig"`     // Ed25519 signature over signedRegion(...)
}

// signedRegion returns the exact bytes that are signed and verified: a small canonical header
// (version, algorithm, key id) followed by the raw payload bytes. Signing the payload bytes
// verbatim - rather than re-marshaling the message - avoids any field-ordering ambiguity.
func signedRegion(version int, alg, kid string, payload []byte) []byte {
	header := fmt.Sprintf("%d\n%s\n%s\n", version, alg, kid)
	region := make([]byte, 0, len(header)+len(payload))
	region = append(region, header...)
	region = append(region, payload...)
	return region
}

// Verification error sentinels. Consumers treat all of these the same way (ack-and-drop, never
// retry - a forged or malformed message will never become valid), but they are distinct so the
// cause can be logged and metered.
var (
	ErrUnsigned           = errors.New("message is not signed")
	ErrUnsupportedVersion = errors.New("unsupported envelope version")
	ErrUnsupportedAlg     = errors.New("unsupported signature algorithm")
	ErrUnknownKey         = errors.New("unknown signing key id")
	ErrBadSignature       = errors.New("signature verification failed")
)

// Metrics. Registered by the daemon (the only executable that verifies).
var (
	VerifyFailCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_msg_verify_fail_count",
		Help: "Capillaries messages rejected by signature verification",
	})
	VerifyUnsignedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_msg_verify_unsigned_count",
		Help: "Capillaries unsigned messages accepted in permissive verify mode",
	})
)
