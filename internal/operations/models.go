package operations

import "time"

// OperationProgress represents the progress of a long-running operation
type OperationProgress struct {
	Stage         string  // Current stage: "scanning", "hashing", "copying", "indexing"
	FilesTotal    int64   // Total number of files to process
	FilesComplete int64   // Number of files completed
	BytesTotal    int64   // Total bytes to process
	BytesComplete int64   // Bytes completed
	CurrentFile   string  // Path of current file being processed
	Percentage    float64 // Completion percentage (0-100)
}

// OperationResult represents the result of a completed operation
type OperationResult struct {
	Success        bool          // Whether operation completed successfully
	Error          error         // Error if operation failed
	FilesProcessed int64         // Number of files processed
	BytesProcessed int64         // Bytes processed
	Duration       time.Duration // Time taken
}

// ProgressCallback is called periodically during long operations
// to report progress updates
type ProgressCallback func(progress OperationProgress)

// CleanResult represents the result of a cleanup operation
type CleanResult struct {
	FilesDeleted int64        // Number of files successfully deleted
	SpaceFreed   int64        // Bytes freed
	Errors       []CleanError // Errors encountered during cleanup
}

// CleanError represents an error during cleanup
type CleanError struct {
	Path  string // File path that failed
	Error error  // The error that occurred
}
