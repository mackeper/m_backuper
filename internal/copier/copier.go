package copier

type Copier interface {
	Copy(src, dst string) (int64, error)
	Close() error
}
