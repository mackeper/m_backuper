package scanner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Path    string
	Size    int64
	ModTime int64 // Unix timestamp
	IsDir   bool
}

type Scanner struct {
	ignorePatterns []string
}

func New(ignorePatterns []string) *Scanner {
	return &Scanner{
		ignorePatterns: ignorePatterns,
	}
}

func (s *Scanner) Scan(paths []string) ([]FileInfo, error) {
	var files []FileInfo
	seen := make(map[string]bool) // Track visited paths to handle symlinks

	for _, path := range paths {
		if err := s.scanPath(path, &files, seen); err != nil {
			return files, err
		}
	}

	return files, nil
}

func (s *Scanner) scanPath(path string, files *[]FileInfo, seen map[string]bool) error {
	// Get absolute path to handle symlinks correctly
	absPath, err := filepath.Abs(path)
	if err != nil {
		slog.Warn("failed to get absolute path", "path", path, "error", err)
		return nil // Continue scanning other paths
	}

	// Check if we've already visited this path (symlink loop detection)
	if seen[absPath] {
		slog.Debug("skipping already visited path", "path", absPath)
		return nil
	}
	seen[absPath] = true

	// Get file info
	info, err := os.Lstat(path) // Use Lstat to not follow symlinks
	if err != nil {
		if os.IsPermission(err) {
			slog.Warn("permission denied", "path", path, "error", err)
			return nil // Continue scanning other paths
		}
		slog.Error("failed to stat file", "path", path, "error", err)
		return nil // Continue scanning other paths
	}

	// Handle symlinks
	if info.Mode()&os.ModeSymlink != 0 {
		// Resolve symlink
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			slog.Warn("failed to resolve symlink", "path", path, "error", err)
			return nil // Skip broken symlinks
		}

		// Get info about the target
		info, err = os.Stat(realPath)
		if err != nil {
			slog.Warn("failed to stat symlink target", "path", realPath, "error", err)
			return nil
		}

		// Use the real path for further processing
		path = realPath
		absPath = realPath
	}

	// Check if path should be ignored
	if s.shouldIgnore(path) {
		slog.Debug("ignoring path", "path", path)
		return nil
	}

	// If it's a directory, walk it
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			if os.IsPermission(err) {
				slog.Warn("permission denied reading directory", "path", path, "error", err)
				return nil
			}
			slog.Error("failed to read directory", "path", path, "error", err)
			return nil
		}

		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			if err := s.scanPath(entryPath, files, seen); err != nil {
				// Log but continue with other entries
				slog.Warn("error scanning path", "path", entryPath, "error", err)
			}
		}
	} else {
		// It's a file, add it to the list
		*files = append(*files, FileInfo{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
			IsDir:   false,
		})
	}

	return nil
}

func (s *Scanner) shouldIgnore(path string) bool {
	for _, pattern := range s.ignorePatterns {
		if s.matchPattern(path, pattern) {
			return true
		}
	}
	return false
}

// Supports patterns like *.tmp, .cache/*, **/node_modules/**
func (s *Scanner) matchPattern(path, pattern string) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)

	// Handle ** patterns (match any number of directories)
	if strings.Contains(pattern, "**") {
		// Convert ** to a regex-like pattern
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]

			// Remove leading/trailing slashes for matching
			prefix = strings.TrimPrefix(prefix, "/")
			suffix = strings.TrimSuffix(suffix, "/")
			suffix = strings.TrimPrefix(suffix, "/")

			if prefix != "" && !strings.Contains(path, prefix) {
				return false
			}
			if suffix != "" && !strings.Contains(path, suffix) {
				return false
			}
			return true
		}
	}

	// Try standard glob matching
	matched, err := filepath.Match(pattern, filepath.Base(path))
	if err == nil && matched {
		return true
	}

	// Try matching against the full path
	matched, err = filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Check if any directory component matches the pattern
	if strings.Contains(pattern, "/") {
		pathParts := strings.Split(path, "/")
		patternParts := strings.Split(pattern, "/")

		// Try to find the pattern in the path
		for i := 0; i <= len(pathParts)-len(patternParts); i++ {
			matches := true
			for j, patternPart := range patternParts {
				matched, err := filepath.Match(patternPart, pathParts[i+j])
				if err != nil || !matched {
					matches = false
					break
				}
			}
			if matches {
				return true
			}
		}
	}

	return false
}

func (s *Scanner) ScanDryRun(paths []string) ([]FileInfo, error) {
	files, err := s.Scan(paths)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	slog.Info("dry-run scan complete", "file_count", len(files))
	for i, file := range files {
		if i < 10 { // Show first 10 files as sample
			slog.Info("would backup", "path", file.Path, "size", file.Size)
		}
	}
	if len(files) > 10 {
		slog.Info("...", "additional_files", len(files)-10)
	}

	return files, nil
}
