package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// CopyJob represents a file copy operation
type CopyJob struct {
	SourcePath string
	DestPath   string
	Size       int64
}

// CopyResult contains the result of a copy operation
type CopyResult struct {
	SourcePath string
	DestPath   string
	Size       int64
	Err        error
}

// Copier copies files with a worker pool
type Copier struct {
	numWorkers int
}

// NewCopier creates a new file copier
func NewCopier(numWorkers int) *Copier {
	if numWorkers <= 0 {
		numWorkers = 2 // Default to 2 workers for I/O operations
	}
	return &Copier{
		numWorkers: numWorkers,
	}
}

// CopyFiles copies multiple files in parallel using a worker pool
func (c *Copier) CopyFiles(ctx context.Context, jobs <-chan CopyJob) <-chan CopyResult {
	results := make(chan CopyResult)

	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < c.numWorkers; i++ {
		wg.Add(1)
		go c.worker(ctx, jobs, results, &wg)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// worker processes copy jobs
func (c *Copier) worker(ctx context.Context, jobs <-chan CopyJob, results chan<- CopyResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			err := c.copyFile(job.SourcePath, job.DestPath)
			results <- CopyResult{
				SourcePath: job.SourcePath,
				DestPath:   job.DestPath,
				Size:       job.Size,
				Err:        err,
			}
		}
	}
}

// copyFile copies a single file, creating directories as needed
func (c *Copier) copyFile(src, dst string) error {
	// Create destination directory if it doesn't exist
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dstDir, err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dstFile.Close()

	// Copy file contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy contents: %w", err)
	}

	// Sync to ensure data is written
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("sync destination: %w", err)
	}

	// Preserve modification time
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		// Non-fatal error, just log it
		return nil
	}

	return nil
}

// CopyFile copies a single file (convenience method)
func (c *Copier) CopyFile(src, dst string) error {
	return c.copyFile(src, dst)
}

// ComputeDestPath computes the destination path for a file
// It preserves the directory structure relative to the source root
func ComputeDestPath(sourcePath, sourceRoot, destRoot string) (string, error) {
	// Get the relative path from source root
	relPath, err := filepath.Rel(sourceRoot, sourcePath)
	if err != nil {
		return "", fmt.Errorf("compute relative path: %w", err)
	}

	// Join with destination root
	destPath := filepath.Join(destRoot, relPath)
	return destPath, nil
}
