package cpu

type CPUWithUsage interface {
	CPUUsage() (float64, error)
}
