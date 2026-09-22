package sc

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// fetch policy-related constants, not to be confused with a similar set in xfer
const urlSchemeFile string = "file"
const urlSchemeHttp string = "http"
const urlSchemeHttps string = "https"
const urlSchemeSftp string = "sftp"
const urlSchemeS3 string = "s3"

// FetchPolicy gates which URLs the framework is willing to fetch for top-level, externally
// supplied resources - specifically the script URL and script-params URL submitted with a run.
//
// Without it, a caller who can influence a submitted script/params URL can make the daemon or
// webapi read arbitrary local files (via the "file" scheme or a bare local path with an empty
// scheme) or reach internal-only network endpoints (cloud metadata services, localhost, private
// ranges) over http(s) - a classic SSRF and local-file-disclosure vector.
//
// The policy is opt-in: a nil *FetchPolicy, or a policy with an empty AllowedSchemes list, allows
// everything (legacy behavior). Operators harden a deployment by populating AllowedSchemes (and,
// for http(s), leaving AllowPrivateHosts false).
type FetchPolicy struct {
	// AllowedSchemes is the set of URL schemes permitted for top-level fetches, e.g. ["https","s3"].
	// A bare local path (empty scheme) and the "file" scheme are both treated as "file".
	// When empty, the policy is disabled and every scheme is allowed.
	AllowedSchemes []string `json:"allowed_schemes" env:"CAPI_FETCH_ALLOWED_SCHEMES, overwrite"`
	// AllowPrivateHosts, when false, blocks http(s) fetches whose host resolves to a loopback,
	// link-local, private, or unspecified IP address. Ignored when the policy is disabled.
	AllowPrivateHosts bool `json:"allow_private_hosts" env:"CAPI_FETCH_ALLOW_PRIVATE_HOSTS, overwrite"`
}

// CheckUrl returns an error if fetching fileUrl is not permitted by this policy.
// A nil policy (or one with no AllowedSchemes) permits everything.
//
// Note: for http(s) the host is resolved and its addresses are checked here, before the fetch.
// This is a best-effort SSRF mitigation and does not by itself close a DNS-rebinding TOCTOU gap
// between this check and the subsequent connection; it blocks the common static-target case.
func (fp *FetchPolicy) CheckUrl(fileUrl string) error {
	if fp == nil || len(fp.AllowedSchemes) == 0 {
		return nil
	}

	u, err := url.Parse(fileUrl)
	if err != nil {
		return fmt.Errorf("cannot parse url %s: %s", fileUrl, err.Error())
	}

	scheme := u.Scheme
	if scheme == "" {
		scheme = urlSchemeFile
	}

	allowed := false
	for _, s := range fp.AllowedSchemes {
		if strings.EqualFold(strings.TrimSpace(s), scheme) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("fetching url scheme %q is not allowed by fetch policy (allowed schemes: %v)", scheme, fp.AllowedSchemes)
	}

	if !fp.AllowPrivateHosts && (scheme == urlSchemeHttp || scheme == urlSchemeHttps || scheme == urlSchemeSftp) {
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
