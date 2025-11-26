package backup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileInfo contains information about a discovered file
type FileInfo struct {
	Path    string
	Size    int64
	ModTime int64
	IsDir   bool
}

// Walker walks a directory tree and discovers files
type Walker struct {
	excludePatterns []string
}

// NewWalker creates a new directory walker
func NewWalker(excludePatterns []string) *Walker {
	return &Walker{
		excludePatterns: excludePatterns,
	}
}

// Walk recursively walks a directory and sends discovered files to the output channel
func (w *Walker) Walk(ctx context.Context, root string) (<-chan FileInfo, <-chan error) {
	files := make(chan FileInfo)
	errs := make(chan error, 1)

	go func() {
		defer close(files)
		defer close(errs)

		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// Log error but continue walking
				select {
				case errs <- fmt.Errorf("walk %s: %w", path, err):
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil // Continue walking
			}

			// Check context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Check if path should be excluded
			if w.shouldExclude(path, d.IsDir()) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			// Skip directories in output (we only want files)
			if d.IsDir() {
				return nil
			}

			// Get file info
			info, err := d.Info()
			if err != nil {
				select {
				case errs <- fmt.Errorf("get info for %s: %w", path, err):
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			// Send file info
			fileInfo := FileInfo{
				Path:    path,
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
				IsDir:   false,
			}

			select {
			case files <- fileInfo:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil && err != context.Canceled {
			select {
			case errs <- err:
			default:
			}
		}
	}()

	return files, errs
}

// shouldExclude checks if a path matches any exclusion pattern
func (w *Walker) shouldExclude(path string, isDir bool) bool {
	for _, pattern := range w.excludePatterns {
		// Try matching against the full path
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}

		// For directory patterns like "node_modules/**"
		if isDir {
			dirName := filepath.Base(path)
			// Remove /** suffix if present
			cleanPattern := pattern
			if len(pattern) > 3 && pattern[len(pattern)-3:] == "/**" {
				cleanPattern = pattern[:len(pattern)-3]
			}

			matched, err := filepath.Match(cleanPattern, dirName)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// WalkMultiple walks multiple root directories and merges the results
func (w *Walker) WalkMultiple(ctx context.Context, roots []string) (<-chan FileInfo, <-chan error) {
	files := make(chan FileInfo)
	errs := make(chan error)

	go func() {
		defer close(files)
		defer close(errs)

		for _, root := range roots {
			// Check if root exists
			if _, err := os.Stat(root); err != nil {
				select {
				case errs <- fmt.Errorf("source %s: %w", root, err):
				case <-ctx.Done():
					return
				}
				continue
			}

			// Walk this root
			rootFiles, rootErrs := w.Walk(ctx, root)

			// Forward all files and errors
			for {
				select {
				case <-ctx.Done():
					return
				case f, ok := <-rootFiles:
					if !ok {
						rootFiles = nil
						if rootErrs == nil {
							goto nextRoot
						}
						continue
					}
					select {
					case files <- f:
					case <-ctx.Done():
						return
					}
				case e, ok := <-rootErrs:
					if !ok {
						rootErrs = nil
						if rootFiles == nil {
							goto nextRoot
						}
						continue
					}
					select {
					case errs <- e:
					case <-ctx.Done():
						return
					}
				}
			}
		nextRoot:
		}
	}()

	return files, errs
}
