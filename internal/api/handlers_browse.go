package api

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	core "github.com/kushiemoon-dev/flacidal-core"
)

const (
	maxBrowseEntries = 2000
	maxBrowseFlac    = 10000
)

type browseEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// handleBrowse lists one folder for the web file picker. scope=tools stays
// inside the allowed roots and can list .flac files; scope=folders lists
// folders only, anywhere on the filesystem (used to pick a download folder or
// library, validated again on save).
func (s *Server) handleBrowse(c *fiber.Ctx) error {
	scope := c.Query("scope")
	path := c.Query("path")
	withFiles := c.Query("files") == "1"

	switch scope {
	case "tools":
		if path == "" {
			path = s.allowedRoots()[0]
		}
		if err := s.confine(path); err != nil {
			return pathError(c, err)
		}
	case "folders":
		withFiles = false
		if path == "" {
			path = homeOrRoot()
		}
		if err := rejectBadPath(path); err != nil {
			return pathError(c, err)
		}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "scope must be tools or folders"})
	}

	path = filepath.Clean(path)
	if scope == "folders" {
		if err := checkReadableDir(path); err != nil {
			return browseError(c, err)
		}
	}
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return browseError(c, err)
	}

	entries := make([]browseEntry, 0, len(dirEntries))
	for _, d := range dirEntries {
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		isDir := d.IsDir()
		if d.Type()&fs.ModeSymlink != 0 {
			if scope == "tools" && s.confine(full) != nil {
				continue
			}
			info, err := os.Stat(full)
			if err != nil {
				continue
			}
			isDir = info.IsDir()
		}
		if !isDir && !(withFiles && strings.EqualFold(filepath.Ext(name), ".flac")) {
			continue
		}
		entries = append(entries, browseEntry{Name: name, Path: full, IsDir: isDir})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	truncated := len(entries) > maxBrowseEntries
	if truncated {
		entries = entries[:maxBrowseEntries]
	}

	return c.JSON(fiber.Map{
		"path":      path,
		"parent":    s.browseParent(scope, path),
		"entries":   entries,
		"truncated": truncated,
	})
}

func (s *Server) browseParent(scope, path string) string {
	parent := filepath.Dir(path)
	if parent == path {
		return ""
	}
	if scope == "tools" && s.confine(parent) != nil {
		return ""
	}
	return parent
}

func homeOrRoot() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	return string(filepath.Separator)
}

func checkReadableDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errNotDir
	}
	return nil
}

var errNotDir = errors.New("path is not a folder")

func browseError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, fs.ErrNotExist):
		status = fiber.StatusNotFound
	case errors.Is(err, fs.ErrPermission):
		status = fiber.StatusForbidden
	case errors.Is(err, errNotDir):
		status = fiber.StatusBadRequest
	}
	msg := "cannot list folder"
	switch status {
	case fiber.StatusNotFound:
		msg = "folder not found"
	case fiber.StatusForbidden:
		msg = "permission denied"
	case fiber.StatusBadRequest:
		msg = errNotDir.Error()
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// handleBrowseFlac returns every .flac under a confined folder, recursively,
// as the absolute paths the other routes expect.
func (s *Server) handleBrowseFlac(c *fiber.Ctx) error {
	path := c.Query("path")
	if err := s.confine(path); err != nil {
		return pathError(c, err)
	}
	files, err := core.ListFLACFiles(path)
	if err != nil {
		return browseError(c, err)
	}
	paths := make([]string, 0, len(files))
	truncated := false
	for _, f := range files {
		if s.confine(f.Path) != nil {
			continue
		}
		if len(paths) == maxBrowseFlac {
			truncated = true
			break
		}
		paths = append(paths, f.Path)
	}
	return c.JSON(fiber.Map{"paths": paths, "truncated": truncated})
}
