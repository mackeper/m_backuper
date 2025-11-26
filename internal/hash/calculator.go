package hash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
)

// Calculator computes SHA256 hashes for files using a worker pool
type Calculator struct {
	numWorkers int
}

// NewCalculator creates a new hash calculator
// If numWorkers is 0, it uses runtime.NumCPU()
func NewCalculator(numWorkers int) *Calculator {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	return &Calculator{
		numWorkers: numWorkers,
	}
}

// HashJob represents a file to be hashed
type HashJob struct {
	Path string
	Size int64
}

// HashResult contains the hash result or error
type HashResult struct {
	Path string
	Hash string
	Size int64
	Err  error
}

// HashFiles hashes multiple files in parallel using a worker pool
func (c *Calculator) HashFiles(ctx context.Context, jobs <-chan HashJob) <-chan HashResult {
	results := make(chan HashResult)

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

// worker processes hash jobs from the jobs channel
func (c *Calculator) worker(ctx context.Context, jobs <-chan HashJob, results chan<- HashResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			hash, err := c.hashFile(job.Path)
			results <- HashResult{
				Path: job.Path,
				Hash: hash,
				Size: job.Size,
				Err:  err,
			}
		}
	}
}

// hashFile computes the SHA256 hash of a single file
func (c *Calculator) hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()

	// Stream file in chunks to handle large files efficiently
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashFile computes the SHA256 hash of a single file (convenience method)
func (c *Calculator) HashFile(path string) (string, error) {
	return c.hashFile(path)
}

// HashPartial computes a partial hash of the first N bytes of a file
// This is useful for quick duplicate detection before computing full hashes
func (c *Calculator) HashPartial(path string, bytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()

	// Read only the first N bytes
	if _, err := io.CopyN(h, f, bytes); err != nil && err != io.EOF {
		return "", fmt.Errorf("read file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
