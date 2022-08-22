package healthcheck

import (
	"time"

	"github.com/vmware/govmomi/vim25/mo"
)

// Package consists of API, that provides info about the Health of the Virtual Machine Server

type CPUInfo struct {
	// Reprensents Current CPU Info of the Virtual Machine Server
	CpuNums        int32 `json:"CpuNums"`
	CpuUsage       int32 `json:"CpuUsage"`
	CpuReservation int32 `json:"CpuReservation"`
}

func NewCPUInfo(CpuUsage int32, CpuNums int32, CpuReservation int32) *CPUInfo {
	return &CPUInfo{
		CpuNums:        CpuNums,
		CpuUsage:       CpuUsage,
		CpuReservation: CpuReservation,
	}
}

type MemoryUsageInfo struct {
	// Represents Current Memory state of the Virtual Machine Server
	MemoryInUse int32 `json:"MemoryInUse"`
	MemoryLeft  int64 `json"MemoryLeft"`
}

func NewMemoryUsageInfo(MemoryInUse int32, MemoryLeft int64) *MemoryUsageInfo {
	return &MemoryUsageInfo{
		MemoryInUse: MemoryInUse,
		MemoryLeft:  MemoryLeft,
	}
}

type AliveInfo struct {
	// Represents Current State of the Virtual Machine
	ConnectionState string `json:"ConnectionState"`
	PowerState      string `json:"PowerState"`
	BootTime        time.Time
}

func NewAliveInfo(ConnectionState string, PowerState string, BootTime time.Time) *AliveInfo {
	return &AliveInfo{
		ConnectionState: ConnectionState,
		PowerState:      PowerState,
		BootTime:        BootTime,
	}
}

type StorageInfo struct {
	UnShared    int64 `json:"Shared"`
	Committed   int64 `json:"Committed"`
	Uncommitted int64 `json:"Uncommitted"`
}

func NewStorageInfo(UnShared int64, Committed int64, Uncommitted int64) *StorageInfo {
	return &StorageInfo{
		UnShared:    UnShared,
		Committed:   Committed,
		Uncommitted: Uncommitted,
	}
}

type VirtualMachineHealthCheckManager struct {
	VirtualMachine *mo.VirtualMachine
}

func NewVirtualMachineHealthCheckManager() *VirtualMachineHealthCheckManager {
	return &VirtualMachineHealthCheckManager{}
}

func (this *VirtualMachineHealthCheckManager) GetCpuMetrics() CPUInfo {
	CpuUsageInfo := this.VirtualMachine.Summary.Runtime.MaxCpuUsage
	CpuNumsInfo := this.VirtualMachine.Summary.Config.NumCpu
	CpuReservation := this.VirtualMachine.Summary.Config.CpuReservation
	newCpuInfo := NewCPUInfo(CpuUsageInfo, CpuNumsInfo, CpuReservation)
	return *newCpuInfo
}

func (this *VirtualMachineHealthCheckManager) GetAliveMetrics() AliveInfo {
	ConnectionState := this.VirtualMachine.Summary.Runtime.ConnectionState
	PowerState := this.VirtualMachine.Summary.Runtime.PowerState
	BootTime := this.VirtualMachine.Summary.Runtime.BootTime
	NewAliveMetric := NewAliveInfo(string(ConnectionState), string(PowerState), *BootTime)
	return *NewAliveMetric
}

func (this *VirtualMachineHealthCheckManager) GetMemoryUsageMetrics() MemoryUsageInfo {
	MemoryUsage := this.VirtualMachine.Summary.Runtime.MaxMemoryUsage
	MemoryOverhead := this.VirtualMachine.Summary.Runtime.MemoryOverhead
	NewMemoryMetric := NewMemoryUsageInfo(MemoryUsage, MemoryOverhead)
	return *NewMemoryMetric
}

func (this *VirtualMachineHealthCheckManager) GetStorageUsageMetrics() StorageInfo {
	StorageShared := this.VirtualMachine.Summary.Storage.Unshared
	StorageCommitted := this.VirtualMachine.Summary.Storage.Committed
	StorageUnCommitted := this.VirtualMachine.Summary.Storage.Uncommitted
	NewStorageMetric := NewStorageInfo(StorageShared, StorageCommitted, StorageUnCommitted)
	return *NewStorageMetric
}
