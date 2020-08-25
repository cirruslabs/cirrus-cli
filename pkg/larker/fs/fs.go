package fs

type FileSystem interface {
	Get(path string) ([]byte, error)
}
