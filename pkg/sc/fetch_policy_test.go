package sc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCheckUrl exercises the FetchPolicy value space: the disabled (allow-everything) modes,
// scheme allow-listing, the empty/"file" scheme aliasing, and the http(s) private-host guard.
//
// The host-resolution cases deliberately use IP *literals* as hosts: net.LookupIP short-circuits
// literals and returns them without issuing a DNS query, so these cases are deterministic and do
// not touch the network. Do not switch them to hostnames - that would make the test flaky and
// dependent on the resolver.
func TestCheckUrl(t *testing.T) {
	type testCase struct {
		name      string
		policy    *FetchPolicy
		url       string
		expectErr bool
	}

	allSchemes := []string{urlSchemeFile, urlSchemeHttp, urlSchemeHttps, urlSchemeS3, urlSchemeSftp}

	cases := []testCase{
		// --- Disabled policy: allows everything (legacy behavior) ---
		{
			name:      "nil policy allows any url",
			policy:    nil,
			url:       "http://10.0.0.1/secret",
			expectErr: false,
		},
		{
			name:      "empty allowed schemes disables the policy",
			policy:    &FetchPolicy{AllowedSchemes: nil, AllowPrivateHosts: false},
			url:       "file:///etc/passwd",
			expectErr: false,
		},
		{
			name:      "empty allowed schemes ignores private hosts too",
			policy:    &FetchPolicy{AllowedSchemes: []string{}, AllowPrivateHosts: false},
			url:       "http://127.0.0.1/metadata",
			expectErr: false,
		},

		// --- Scheme allow-listing ---
		{
			name:      "explicit file scheme allowed",
			policy:    &FetchPolicy{AllowedSchemes: []string{urlSchemeFile}},
			url:       "file:///tmp/script.json",
			expectErr: false,
		},
		{
			name:      "empty scheme is treated as file and allowed",
			policy:    &FetchPolicy{AllowedSchemes: []string{urlSchemeFile}},
			url:       "/tmp/script.json",
			expectErr: false,
		},
		{
			name:      "empty scheme blocked when file not allowed",
			policy:    &FetchPolicy{AllowedSchemes: []string{urlSchemeHttps}},
			url:       "/etc/passwd",
			expectErr: true,
		},
		{
			name:      "scheme not in allow-list is blocked",
			policy:    &FetchPolicy{AllowedSchemes: []string{urlSchemeHttps}},
			url:       "sftp://host/path",
			expectErr: true,
		},
		{
			name:      "s3 scheme allowed",
			policy:    &FetchPolicy{AllowedSchemes: []string{urlSchemeS3}},
			url:       "s3://bucket/key",
			expectErr: false,
		},
		{
			name:      "scheme match is case-insensitive and trims whitespace",
			policy:    &FetchPolicy{AllowedSchemes: []string{" HTTPS "}, AllowPrivateHosts: true},
			url:       "https://example.com/script.json",
			expectErr: false,
		},

		// --- http(s) private-host guard, AllowPrivateHosts = false ---
		{
			name:      "public host allowed over https",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "https://8.8.8.8/script.json",
			expectErr: false,
		},
		{
			name:      "loopback host blocked",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "http://127.0.0.1/metadata",
			expectErr: true,
		},
		{
			name:      "private 10.x host blocked",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "http://10.0.0.1/",
			expectErr: true,
		},
		{
			name:      "private 192.168.x host blocked",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "https://192.168.1.1/",
			expectErr: true,
		},
		{
			name:      "link-local host blocked (cloud metadata range)",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "http://169.254.169.254/latest/meta-data/",
			expectErr: true,
		},
		{
			name:      "unspecified host blocked",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "http://0.0.0.0/",
			expectErr: true,
		},
		{
			name:      "empty host blocked",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "http:///path",
			expectErr: true,
		},

		// --- AllowPrivateHosts = true relaxes the host guard ---
		{
			name:      "private host allowed when AllowPrivateHosts is true",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: true},
			url:       "http://127.0.0.1/metadata",
			expectErr: false,
		},

		// --- Host guard applies only to http(s) ---
		{
			name:      "s3 with private-looking host is not host-checked",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "s3://127.0.0.1/bucket/key",
			expectErr: false,
		},
		{
			name:      "file scheme is not host-checked",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes, AllowPrivateHosts: false},
			url:       "file:///etc/hosts",
			expectErr: false,
		},

		// --- Unparseable URL ---
		{
			name:      "unparseable url is rejected",
			policy:    &FetchPolicy{AllowedSchemes: allSchemes},
			url:       "http://%zz",
			expectErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.policy.CheckUrl(tc.url)
			if tc.expectErr {
				assert.Error(t, err, "expected CheckUrl(%q) to return an error", tc.url)
			} else {
				assert.NoError(t, err, "expected CheckUrl(%q) to succeed", tc.url)
			}
		})
	}
}
