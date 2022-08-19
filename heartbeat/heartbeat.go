package heartbeat

// Package offers a number of different Classes, that is used for Perform Healthchecks
// Onto the Virtual Machine

type VirtualMachineHeartbeatMetric struct {
}

func NewVirtualMachineHeartBeatMetric() *VirtualMachineHeartbeatMetric {
	return &VirtualMachineHeartbeatMetric{}
}

type VirtualMachineHeartBeatManager struct {
}

func NewVirtualMachineHeartBeatManager() *VirtualMachineHeartBeatManager {
	return &VirtualMachineHeartBeatManager{}
}

func (this *VirtualMachineHeartBeatManager) GetCpuMetrics() {

}

func (this *VirtualMachineHeartBeatManager) GetMemoryUsageMetrics() {

}

func (this *VirtualMachineHeartBeatManager) GetMetrics() {
	
}
