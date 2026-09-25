package sc

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// fetch policy-related constants, not to be confused with a similar set in xfer
const FetchUrlSchemeFile string = "file"
const FetchUrlSchemeHttp string = "http"
const FetchUrlSchemeHttps string = "https"
const FetchUrlSchemeSftp string = "sftp"
const FetchUrlSchemeS3 string = "s3"

// FetchPolicy gates which URLs the framework is willing to fetch for externally supplied resources:
// the top-level script URL and script-params URL submitted with a run, and the file URLs embedded in
// a script (file_reader "urls" and file_creator "url_template").
//
// Without gating, a caller who can influence any of those URLs can make the daemon or webapi read
// arbitrary local files (via the "file" scheme or a bare local path with an empty scheme) or reach
// internal-only network endpoints (cloud metadata services, localhost, private ranges) over http(s) -
// a classic SSRF and local-file-disclosure vector.
//
// The policy is default-deny for the dangerous cases: when it is unconfigured (a nil *FetchPolicy or
// an empty AllowedSchemes list) a safe built-in default applies - only https and s3 are permitted,
// the "file"/empty scheme (local reads) is denied, and http(s)/sftp hosts that resolve to private,
// loopback, link-local, or unspecified addresses are blocked. Operators who need other schemes
// (notably "file" for local scripts/inputs, e.g. the toolbelt, or "http"/"sftp") must opt in by
// populating AllowedSchemes explicitly; doing so replaces the default entirely.
type FetchPolicy struct {
	// AllowedSchemes is the set of URL schemes permitted for fetches, e.g. ["https","s3","file"].
	// A bare local path (empty scheme) and the "file" scheme are both treated as "file".
	// When empty, the safe built-in default (DefaultAllowedSchemes) applies instead.
	AllowedSchemes []string `json:"allowed_schemes" env:"CAPI_FETCH_ALLOWED_SCHEMES, overwrite"`
	// AllowPrivateHosts, when false, blocks http(s)/sftp fetches whose host resolves to a loopback,
	// link-local, private, or unspecified IP address. Only honored when AllowedSchemes is configured;
	// the built-in default always blocks private hosts.
	AllowPrivateHosts bool `json:"allow_private_hosts" env:"CAPI_FETCH_ALLOW_PRIVATE_HOSTS, overwrite"`
}

// DefaultAllowedSchemes is applied when an operator has not configured AllowedSchemes. It permits
// only remote fetches that are safe by default (https, s3); combined with the private-host block it
// denies the two dangerous defaults - local file reads (file/empty scheme) and SSRF to internal
// endpoints. Operators needing file:// or sftp must set AllowedSchemes explicitly.
var DefaultAllowedSchemes = []string{FetchUrlSchemeHttps, FetchUrlSchemeS3}

// CheckUrl returns an error if fetching fileUrl is not permitted by this policy. When the policy is
// unconfigured (nil, or no AllowedSchemes), the safe built-in default (DefaultAllowedSchemes, private
// hosts blocked) is enforced rather than allowing everything.
//
// Note: for http(s)/sftp the host is resolved and its addresses are checked here, before the fetch.
// This is a best-effort SSRF mitigation and does not by itself close a DNS-rebinding TOCTOU gap
// between this check and the subsequent connection; it blocks the common static-target case.
func (fp *FetchPolicy) CheckUrl(fileUrl string) error {
	// Effective policy: an explicitly configured AllowedSchemes replaces the default entirely;
	// otherwise fall back to the safe built-in default with private hosts blocked.
	allowedSchemes := DefaultAllowedSchemes
	allowPrivateHosts := false
	if fp != nil && len(fp.AllowedSchemes) > 0 {
		allowedSchemes = fp.AllowedSchemes
		allowPrivateHosts = fp.AllowPrivateHosts
	}

	u, err := url.Parse(fileUrl)
	if err != nil {
		return fmt.Errorf("cannot parse url %s: %s", fileUrl, err.Error())
	}

	scheme := u.Scheme
	if scheme == "" {
		scheme = FetchUrlSchemeFile
	}

	allowed := false
	for _, s := range allowedSchemes {
		if strings.EqualFold(strings.TrimSpace(s), scheme) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("fetching url scheme %q is not allowed by fetch policy (allowed schemes: %v)", scheme, allowedSchemes)
	}

	if !allowPrivateHosts && (scheme == FetchUrlSchemeHttp || scheme == FetchUrlSchemeHttps || scheme == FetchUrlSchemeSftp) {
		if err := checkHostIsPublic(u.Hostname()); err != nil {
			return err
		}
	}

	return nil
}

func checkHostIsPublic(host string) error {
	if host == "" {
		return errors.New("fetch policy: url has an empty host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("fetch policy: cannot resolve host %s: %s", host, err.Error())
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
			return fmt.Errorf("fetch policy: host %s resolves to non-public address %s, blocked to prevent SSRF", host, ip.String())
		}
	}
	return nil
}
