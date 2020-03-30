package subsystems

type ResourceConfig struct {
	MemoryLimit string
	CpuPeriod   string
	CpuQuota    string
	CpuSet      string
}

type Subsystem interface {
	Name() string
	Set(path string, res *ResourceConfig) error
	Apply(path string, pid int) error
	Remove(path string) error
}

var (
	SubsystemsIns = []Subsystem{
		&CpuLimitSubSystem{},
		&MemorySubSystem{},
		&CpuSubSystem{},
	}
)
