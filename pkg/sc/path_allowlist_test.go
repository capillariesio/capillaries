package sc

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustMkdir creates dir (and parents) or fails the test.
func mustMkdir(t *testing.T, dir string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return dir
}

// mustWrite creates a file with some content or fails the test.
func mustWrite(t *testing.T, path string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
	return path
}

// TestCheckPathAllowed_Disabled verifies the feature is opt-in: an empty allowlist imposes no
// restriction on either inputs or outputs, regardless of scheme (including otherwise-dangerous ones).
func TestCheckPathAllowed_Disabled(t *testing.T) {
	for _, u := range []string{
		"/etc/passwd",
		"file:///etc/shadow",
		"https://anywhere.example.com/x",
		"s3://any-bucket/any/key",
		"sftp://host/x",
	} {
		assert.NoError(t, CheckInputPathAllowed(nil, u), "empty input allowlist should allow %s", u)
		assert.NoError(t, CheckOutputPathAllowed([]string{}, u), "empty output allowlist should allow %s", u)
	}
}

// TestCheckInputPathAllowed_Local exercises local (bare-path) input confinement, including the
// path-traversal escape and the sibling-prefix boundary case.
func TestCheckInputPathAllowed_Local(t *testing.T) {
	base := t.TempDir()
	inDir := mustMkdir(t, filepath.Join(base, "inputs"))
	nested := mustMkdir(t, filepath.Join(inDir, "sub"))
	okFile := mustWrite(t, filepath.Join(inDir, "data.csv"))
	nestedFile := mustWrite(t, filepath.Join(nested, "deep.csv"))

	// A sibling directory that shares a textual prefix with inDir ("inputs" vs "inputs-secret").
	siblingDir := mustMkdir(t, filepath.Join(base, "inputs-secret"))
	siblingFile := mustWrite(t, filepath.Join(siblingDir, "leak.csv"))

	// A file entirely outside the allowed dir.
	outsideFile := mustWrite(t, filepath.Join(base, "outside.csv"))

	allow := []string{inDir}

	assert.NoError(t, CheckInputPathAllowed(allow, okFile), "file directly in allowed dir")
	assert.NoError(t, CheckInputPathAllowed(allow, nestedFile), "file in nested subdir of allowed dir")

	assert.Error(t, CheckInputPathAllowed(allow, outsideFile), "file outside allowed dir must be denied")
	assert.Error(t, CheckInputPathAllowed(allow, siblingFile), "sibling-prefix dir must not be treated as contained")
	assert.Error(t, CheckInputPathAllowed(allow, filepath.Join(inDir, "..", "outside.csv")), "traversal escape must be denied")
}

// TestCheckOutputPathAllowed_Local exercises local output confinement. Unlike inputs, the output file
// does not exist yet, so containment is decided from the (existing) parent directory.
func TestCheckOutputPathAllowed_Local(t *testing.T) {
	base := t.TempDir()
	outDir := mustMkdir(t, filepath.Join(base, "outputs"))
	nested := mustMkdir(t, filepath.Join(outDir, "run00001"))

	allow := []string{outDir}

	// Non-existent leaf files whose parent dir is inside the allowed dir.
	assert.NoError(t, CheckOutputPathAllowed(allow, filepath.Join(outDir, "result.csv")), "new file in allowed dir")
	assert.NoError(t, CheckOutputPathAllowed(allow, filepath.Join(nested, "part-0.csv")), "new file in allowed nested dir")

	// Escapes.
	assert.Error(t, CheckOutputPathAllowed(allow, filepath.Join(outDir, "..", "evil.csv")), "traversal escape must be denied")
	assert.Error(t, CheckOutputPathAllowed(allow, filepath.Join(base, "elsewhere", "x.csv")), "output outside allowed dir must be denied")
}

// TestCheckPathAllowed_LocalSymlinkEscape verifies that a symlink planted inside an allowed dir but
// pointing outside it (a classic escape) is rejected, for both a symlinked leaf file (input) and a
// symlinked leaf that an output os.Create would follow.
func TestCheckPathAllowed_LocalSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is unreliable on Windows CI")
	}
	base := t.TempDir()
	inDir := mustMkdir(t, filepath.Join(base, "inputs"))
	outside := mustMkdir(t, filepath.Join(base, "outside"))
	secret := mustWrite(t, filepath.Join(outside, "secret.csv"))

	// Symlinked file inside inDir -> outside/secret.csv.
	fileLink := filepath.Join(inDir, "link.csv")
	require.NoError(t, os.Symlink(secret, fileLink))
	assert.Error(t, CheckInputPathAllowed([]string{inDir}, fileLink), "symlinked leaf escaping the allowed dir must be denied (input)")
	assert.Error(t, CheckOutputPathAllowed([]string{inDir}, fileLink), "existing symlinked leaf that would be followed on write must be denied (output)")

	// Symlinked directory inside inDir -> outside; a file "under" it resolves outside.
	dirLink := filepath.Join(inDir, "linkdir")
	require.NoError(t, os.Symlink(outside, dirLink))
	assert.Error(t, CheckInputPathAllowed([]string{inDir}, filepath.Join(dirLink, "secret.csv")), "file under a symlinked-out dir must be denied")

	// Sanity: a real (non-symlinked) file inside the allowed dir is still permitted.
	realcsv := mustWrite(t, filepath.Join(inDir, "real.csv"))
	assert.NoError(t, CheckInputPathAllowed([]string{inDir}, realcsv))
}

// TestCheckPathAllowed_FileSchemeEntry verifies that a local base dir written as a file:// URL entry
// is honored (not skipped as if it were a remote entry).
func TestCheckPathAllowed_FileSchemeEntry(t *testing.T) {
	base := t.TempDir()
	inDir := mustMkdir(t, filepath.Join(base, "inputs"))
	okFile := mustWrite(t, filepath.Join(inDir, "data.csv"))

	allow := []string{"file://" + inDir}

	// Target given as a bare path.
	assert.NoError(t, CheckInputPathAllowed(allow, okFile), "file:// entry should confine a bare-path target")
	// Target given as a file:// URL.
	assert.NoError(t, CheckInputPathAllowed(allow, "file://"+okFile), "file:// entry should confine a file:// target")
	// Escape still denied.
	assert.Error(t, CheckInputPathAllowed(allow, filepath.Join(base, "outside.csv")))
}

// TestCheckPathAllowed_Remote exercises remote-prefix matching: scheme+host equality and boundary-
// aware path containment, across https/s3/sftp.
func TestCheckPathAllowed_Remote(t *testing.T) {
	allow := []string{
		"https://data.example.com/in/",
		"s3://corp-lake/raw/",
		"sftp://sftp.example.com/incoming/",
		"https://whole-host.example.com", // empty path => whole host allowed
	}

	cases := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{"https within prefix", "https://data.example.com/in/day=1/f.csv", false},
		{"https prefix path exactly", "https://data.example.com/in", false},
		{"https host-prefix trick denied", "https://data.example.com/in-secret/f.csv", true},
		{"https different path denied", "https://data.example.com/out/f.csv", true},
		{"https different host denied", "https://evil.example.com/in/f.csv", true},
		{"https host case-insensitive", "https://Data.Example.com/in/f.csv", false},
		{"scheme mismatch denied", "http://data.example.com/in/f.csv", true},
		{"s3 within prefix", "s3://corp-lake/raw/2026/f.parquet", false},
		{"s3 wrong bucket denied", "s3://other-lake/raw/f.parquet", true},
		{"s3 sibling-prefix denied", "s3://corp-lake/raw-archive/f.parquet", true},
		{"sftp within prefix", "sftp://sftp.example.com/incoming/f.csv", false},
		{"sftp wrong path denied", "sftp://sftp.example.com/private/f.csv", true},
		{"whole-host allows any path", "https://whole-host.example.com/anything/here.csv", false},
		{"whole-host still host-scoped", "https://whole-host2.example.com/x.csv", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errIn := CheckInputPathAllowed(allow, tc.url)
			errOut := CheckOutputPathAllowed(allow, tc.url)
			if tc.expectErr {
				assert.Error(t, errIn, "input %s", tc.url)
				assert.Error(t, errOut, "output %s", tc.url)
			} else {
				assert.NoError(t, errIn, "input %s", tc.url)
				assert.NoError(t, errOut, "output %s", tc.url)
			}
		})
	}
}

// TestCheckPathAllowed_Port verifies host matching is port-sensitive (u.Host includes the port).
func TestCheckPathAllowed_Port(t *testing.T) {
	allow := []string{"https://host.example.com:8443/in/"}
	assert.NoError(t, CheckInputPathAllowed(allow, "https://host.example.com:8443/in/f.csv"), "matching port allowed")
	assert.Error(t, CheckInputPathAllowed(allow, "https://host.example.com/in/f.csv"), "missing port must not match a ported allow entry")
	assert.Error(t, CheckInputPathAllowed(allow, "https://host.example.com:9443/in/f.csv"), "different port must not match")
}

// TestCheckPathAllowed_MixedList verifies a single list can hold both local dirs and remote prefixes:
// a local target only matches local entries and a remote target only matches remote entries.
func TestCheckPathAllowed_MixedList(t *testing.T) {
	base := t.TempDir()
	inDir := mustMkdir(t, filepath.Join(base, "inputs"))
	okFile := mustWrite(t, filepath.Join(inDir, "data.csv"))

	allow := []string{inDir, "https://data.example.com/in/", "s3://corp-lake/raw/"}

	assert.NoError(t, CheckInputPathAllowed(allow, okFile), "local target matches the local entry")
	assert.NoError(t, CheckInputPathAllowed(allow, "https://data.example.com/in/f.csv"), "remote target matches the remote entry")
	assert.NoError(t, CheckInputPathAllowed(allow, "s3://corp-lake/raw/f.parquet"), "s3 target matches the s3 entry")
	assert.Error(t, CheckInputPathAllowed(allow, "https://data.example.com/out/f.csv"), "remote target off-prefix denied")
	assert.Error(t, CheckInputPathAllowed(allow, filepath.Join(base, "outside.csv")), "local target off-dir denied")
}

// TestCheckPathAllowed_KindMismatch verifies that when the allowlist contains only entries of the
// other kind, the target is denied (a remote-only list denies local targets, and vice versa).
func TestCheckPathAllowed_KindMismatch(t *testing.T) {
	base := t.TempDir()
	inDir := mustMkdir(t, filepath.Join(base, "inputs"))
	okFile := mustWrite(t, filepath.Join(inDir, "data.csv"))

	// Local target, remote-only allowlist.
	assert.Error(t, CheckInputPathAllowed([]string{"https://data.example.com/in/"}, okFile))
	// Remote target, local-only allowlist.
	assert.Error(t, CheckInputPathAllowed([]string{inDir}, "https://data.example.com/in/f.csv"))
}

// TestCheckPathAllowed_UnsupportedScheme verifies that an unknown scheme is rejected when the
// allowlist is active (it is neither a recognized local nor remote target).
func TestCheckPathAllowed_UnsupportedScheme(t *testing.T) {
	allow := []string{"https://data.example.com/in/"}
	assert.Error(t, CheckInputPathAllowed(allow, "ftp://data.example.com/in/f.csv"))
	assert.Error(t, CheckOutputPathAllowed(allow, "gopher://data.example.com/x"))
}

// TestCheckPathAllowed_ErrorLabels verifies the direction-specific prefix on error messages, so an
// operator can tell an input violation from an output one.
func TestCheckPathAllowed_ErrorLabels(t *testing.T) {
	base := t.TempDir()
	inDir := mustMkdir(t, filepath.Join(base, "inputs"))
	outsideFile := filepath.Join(base, "outside.csv")

	errIn := CheckInputPathAllowed([]string{inDir}, outsideFile)
	require.Error(t, errIn)
	assert.Contains(t, errIn.Error(), "input")

	errOut := CheckOutputPathAllowed([]string{inDir}, outsideFile)
	require.Error(t, errOut)
	assert.Contains(t, errOut.Error(), "output")
}
