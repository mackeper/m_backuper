package copier

import "io"

type Copier interface {
	Copy(src, dst string) (int64, error)
	Close() error
}

func copyFile(src, dst string, copyFunc func(io.Reader, io.Writer) (int64, error)) (int64, error) {
	// This will be implemented by specific copier implementations
	return 0, nil
}
