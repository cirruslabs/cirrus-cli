package memory

type MemoryWithUsage interface {
	MemoryUsage() (float64, error)
}
