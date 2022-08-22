package healthcheck

// Package consists of API, that provides info about the Health of the Virtual Machine Server

type VirtualMachineHealthCheckManager struct {
}

func NewVirtualMachineHealthCheckManager() *VirtualMachineHealthCheckManager {
	return &VirtualMachineHealthCheckManager{}
}

func (this *VirtualMachineHealthCheckManager) GetCpuMetrics()

func (this *VirtualMachineHealthCheckManager) GetMemoryUsageMetrics()
