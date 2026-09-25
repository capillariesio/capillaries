package sc

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// CheckInputPathAllowed confines a file_reader input to an operator-approved allowlist, closing a
// local-file-disclosure / SSRF-target gap: the script author controls file_reader "urls", so without
// this a run could be steered to read any file or remote object the daemon can reach and surface it
// as table rows. Opt-in and non-breaking: empty allowedPaths returns nil (no restriction).
func CheckInputPathAllowed(allowedPaths []string, fileUrl string) error {
	if err := checkPathAllowed(allowedPaths, fileUrl); err != nil {
		return fmt.Errorf("input %s", err.Error())
	}
	return nil
}

// CheckOutputPathAllowed confines a file_creator output to an operator-approved allowlist, closing a
// path-traversal / arbitrary-file-overwrite gap: the script author controls url_template, so without
// this a run could be steered to write over any file or remote object the daemon can reach. Opt-in
// and non-breaking: empty allowedPaths returns nil (no restriction).
func CheckOutputPathAllowed(allowedPaths []string, fileUrl string) error {
	if err := checkPathAllowed(allowedPaths, fileUrl); err != nil {
		return fmt.Errorf("output %s", err.Error())
	}
	return nil
}

// checkPathAllowed is the shared allowlist check for both input and output file URLs. Each allowlist
// entry is interpreted by its own scheme: a bare path or file:// entry is a local base directory
// (filesystem containment), while an http/https/sftp/s3 entry is a remote prefix (scheme+host must
// match exactly and the target path must be contained within the entry's path). A target only needs
// to match one entry of the same kind. Returns nil when allowedPaths is empty (feature disabled).
func checkPathAllowed(allowedPaths []string, fileUrl string) error {
	if len(allowedPaths) == 0 {
		return nil
	}

	u, err := url.Parse(fileUrl)
	if err != nil {
		return fmt.Errorf("url %s cannot be parsed: %s", fileUrl, err.Error())
	}

	switch strings.ToLower(u.Scheme) {
	case "", FetchUrlSchemeFile:
		// Local: for a bare path fileUrl is the path that os.Open/os.Create receives; for file://
		// use the URL path component.
		localPath := fileUrl
		if strings.EqualFold(u.Scheme, FetchUrlSchemeFile) {
			localPath = u.Path
		}
		return checkLocalPathAllowed(allowedPaths, localPath)
	case FetchUrlSchemeHttp, FetchUrlSchemeHttps, FetchUrlSchemeSftp, FetchUrlSchemeS3:
		return checkRemoteUrlAllowed(allowedPaths, u, fileUrl)
	default:
		return fmt.Errorf("url %s uses unsupported scheme %q", fileUrl, u.Scheme)
	}
}

// checkLocalPathAllowed confines localPath to one of the local base directories in allowedPaths.
// Local entries may be bare paths or file:// URLs; entries with a remote scheme (http/https/sftp/s3)
// do not apply to a local target and are ignored here. Symlinks are resolved to defeat
// symlink-escape tricks: the full path is resolved when it exists (an input, or an output whose leaf
// already exists - e.g. a planted symlink os.Create would follow); when the leaf does not exist yet
// (the normal output case) its parent directory is resolved and the leaf name re-attached.
func checkLocalPathAllowed(allowedPaths []string, localPath string) error {
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("path %s cannot be resolved: %s", localPath, err.Error())
	}

	resolvedPath := absPath
	if p, err := filepath.EvalSymlinks(absPath); err == nil {
		resolvedPath = p
	} else if parent, err := filepath.EvalSymlinks(filepath.Dir(absPath)); err == nil {
		resolvedPath = filepath.Join(parent, filepath.Base(absPath))
	}

	sawLocalEntry := false
	for _, entry := range allowedPaths {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// A local base dir may be written as a bare path or as a file:// URL; a remote-prefix entry
		// (http/https/sftp/s3) does not apply to a local target and is skipped here.
		dir := entry
		if strings.Contains(entry, "://") {
			eu, err := url.Parse(entry)
			if err != nil || !strings.EqualFold(eu.Scheme, FetchUrlSchemeFile) {
				continue
			}
			dir = eu.Path
		}
		sawLocalEntry = true
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if resolvedDir, err := filepath.EvalSymlinks(absDir); err == nil {
			absDir = resolvedDir
		}
		rel, err := filepath.Rel(absDir, resolvedPath)
		if err != nil {
			continue
		}
		// Contained iff rel does not climb out of absDir and is not absolute.
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return nil
		}
	}

	if !sawLocalEntry {
		return fmt.Errorf("local path %s is denied: the allowlist %v permits only remote locations", localPath, allowedPaths)
	}
	return fmt.Errorf("local path %s is not within any allowed directory %v", localPath, allowedPaths)
}

// checkRemoteUrlAllowed confines a remote target URL to one of the remote prefixes in allowedPaths
// (bare/local entries are ignored here). An entry matches when its scheme and host equal the target's
// (case-insensitive) and the target path is contained within the entry's path. Path containment is
// boundary-aware: "https://h/in" matches "https://h/in/x" but not "https://h/in-secret". An entry
// with an empty/"/" path allows the whole host. Query string, fragment, and userinfo are not matched.
func checkRemoteUrlAllowed(allowedPaths []string, u *url.URL, fileUrl string) error {
	targetPath := path.Clean("/" + strings.TrimPrefix(u.Path, "/"))

	sawRemoteEntry := false
	for _, entry := range allowedPaths {
		entry = strings.TrimSpace(entry)
		if entry == "" || !strings.Contains(entry, "://") {
			continue // empty, or a local-directory entry that does not apply to a remote target
		}
		au, err := url.Parse(entry)
		if err != nil || au.Scheme == "" {
			continue
		}
		sawRemoteEntry = true
		if !strings.EqualFold(au.Scheme, u.Scheme) || !strings.EqualFold(au.Host, u.Host) {
			continue
		}
		allowedPath := path.Clean("/" + strings.TrimPrefix(au.Path, "/"))
		if allowedPath == "/" || targetPath == allowedPath || strings.HasPrefix(targetPath, allowedPath+"/") {
			return nil
		}
	}

	if !sawRemoteEntry {
		return fmt.Errorf("url %s is denied: the allowlist %v permits only local directories", fileUrl, allowedPaths)
	}
	return fmt.Errorf("url %s is not within any allowed remote prefix %v", fileUrl, allowedPaths)
}
