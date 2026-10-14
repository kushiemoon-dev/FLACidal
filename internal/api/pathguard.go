package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	core "github.com/kushiemoon-dev/flacidal-core"
)

var (
	errPathInvalid = errors.New("path must be absolute and non-empty")
	errPathOutside = errors.New("path is outside the allowed folders")
)

// allowedRoots reads the live config on every call so a saved config change
// applies immediately.
func (s *Server) allowedRoots() []string {
	download := s.config.DownloadFolder
	if download == "" {
		download = core.GetDefaultDownloadFolder()
	}
	roots := []string{download}
	for _, p := range s.config.ExternalLibraryPaths {
		if p != "" {
			roots = append(roots, p)
		}
	}
	return roots
}

// confine checks path against the download folder and external library paths,
// following symlinks. Callers keep operating on the original (cleaned) path:
// responses reference input paths that the front matches by value.
func (s *Server) confine(path string) error {
	if path == "" || strings.ContainsRune(path, 0) || !filepath.IsAbs(path) {
		return errPathInvalid
	}
	// Clean would fold "link/.." away while the OS resolves the symlink first,
	// so a ".." segment is refused outright instead of normalised.
	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if seg == ".." {
			return errPathOutside
		}
	}
	resolved, err := resolveExisting(filepath.Clean(path))
	if err != nil {
		return errPathOutside
	}
	for _, root := range s.allowedRoots() {
		r, err := resolveExisting(filepath.Clean(root))
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(r, resolved)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil
		}
	}
	return errPathOutside
}

// resolveExisting resolves symlinks on the longest existing prefix of p and
// re-joins the missing tail, so not-yet-created targets are still checked.
func resolveExisting(p string) (string, error) {
	resolved, err := filepath.EvalSymlinks(p)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(p)
	if parent == p {
		return p, nil
	}
	rp, err := resolveExisting(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(rp, filepath.Base(p)), nil
}

func pathError(c *fiber.Ctx, err error) error {
	status := fiber.StatusBadRequest
	if errors.Is(err, errPathOutside) {
		status = fiber.StatusForbidden
	}
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

func validateAbsFolder(p string) (string, error) {
	clean := filepath.Clean(p)
	if !filepath.IsAbs(p) || clean == string(filepath.Separator) {
		return "", fmt.Errorf("folder must be an absolute path other than /")
	}
	return clean, nil
}

// validateDownloadFolder allows empty, which means "use the default".
func validateDownloadFolder(p string) (string, error) {
	if p == "" {
		return "", nil
	}
	return validateAbsFolder(p)
}

// validateLibraryPaths refuses "/" so an authenticated client cannot widen
// the confinement roots to the whole filesystem.
func validateLibraryPaths(ps []string) ([]string, error) {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		clean, err := validateAbsFolder(p)
		if err != nil {
			return nil, fmt.Errorf("invalid library path %q: %w", p, err)
		}
		out = append(out, clean)
	}
	return out, nil
}
