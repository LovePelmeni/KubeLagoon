package healthcheck

import (
	"time"

	"github.com/vmware/govmomi/vim25/mo"
)

// Package consists of API, that provides info about the Health of the Virtual Machine Server

type CPUInfo struct {
	// Reprensents Current CPU Info of the Virtual Machine Server
	OverallCpuUsage    int32 `json:"OverallCpuUsage"`
	OverallCpuReadness int32 `json:"OverallCpuReadness"`
	CpuNums            int32 `json:"CpuNums"`
	MaxCpuUsage        int32 `json:"CpuUsage"`
	CpuReservation     int32 `json:"CpuReservation"`
}

func NewCPUInfo(OverallCpuUsage int32, OverallCpuReadness int32, MaxCpuUsage int32, CpuNums int32, CpuReservation int32) *CPUInfo {
	return &CPUInfo{
		CpuNums:        CpuNums,
		MaxCpuUsage:    MaxCpuUsage,
		CpuReservation: CpuReservation,
	}
}

type MemoryUsageInfo struct {
	// Represents Current Memory state of the Virtual Machine Server
	Shared         int32 `json:"SharedMemory"`
	Granted        int32 `json:"GrantedMemory"`
	MemoryOverhead int64 `json:"MemoryOverhead"`
	MaxMemoryUsage int32 `json:"MaxMemoryUsage"`
}

func NewMemoryUsageInfo(SharedMemory int32, GrantedMemory int32, MemoryOverhead int64, MaxMemoryUsage int32) *MemoryUsageInfo {
	return &MemoryUsageInfo{
		Shared:         SharedMemory,
		Granted:        GrantedMemory,
		MemoryOverhead: MemoryOverhead,
		MaxMemoryUsage: MaxMemoryUsage,
	}
}

type AliveInfo struct {
	// Represents Current State of the Virtual Machine
	OverallStatus   string `json:"OverallStatus"`
	ConnectionState string `json:"ConnectionState"`
	PowerState      string `json:"PowerState"`
	BootTime        time.Time
}

func NewAliveInfo(OverallStatus string, ConnectionState string, PowerState string, BootTime time.Time) *AliveInfo {
	return &AliveInfo{
		OverallStatus:   OverallStatus,
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

	OverallCpuUsage := this.VirtualMachine.Summary.QuickStats.OverallCpuUsage
	OverallCpuReadness := this.VirtualMachine.Summary.QuickStats.OverallCpuReadiness

	MaxCpuUsageInfo := this.VirtualMachine.Summary.Runtime.MaxCpuUsage
	CpuNumsInfo := this.VirtualMachine.Summary.Config.NumCpu
	CpuReservation := this.VirtualMachine.Summary.Config.CpuReservation
	newCpuInfo := NewCPUInfo(OverallCpuUsage, OverallCpuReadness, MaxCpuUsageInfo, CpuNumsInfo, CpuReservation)
	return *newCpuInfo
}

func (this *VirtualMachineHealthCheckManager) GetAliveMetrics() AliveInfo {
	OverallStatus := this.VirtualMachine.Summary.OverallStatus
	ConnectionState := this.VirtualMachine.Summary.Runtime.ConnectionState
	PowerState := this.VirtualMachine.Summary.Runtime.PowerState
	BootTime := this.VirtualMachine.Summary.Runtime.BootTime
	NewAliveMetric := NewAliveInfo(string(OverallStatus), string(ConnectionState), string(PowerState), *BootTime)
	return *NewAliveMetric
}

func (this *VirtualMachineHealthCheckManager) GetMemoryUsageMetrics() MemoryUsageInfo {

	MaxMemoryUsage := this.VirtualMachine.Summary.Runtime.MaxMemoryUsage
	MemoryOverhead := this.VirtualMachine.Summary.Runtime.MemoryOverhead

	OverallSharedMemory := this.VirtualMachine.Summary.QuickStats.SharedMemory
	GrantedMemory := this.VirtualMachine.Summary.QuickStats.GrantedMemory

	NewMemoryMetric := NewMemoryUsageInfo(OverallSharedMemory, GrantedMemory, MemoryOverhead, MaxMemoryUsage)
	return *NewMemoryMetric
}

func (this *VirtualMachineHealthCheckManager) GetStorageUsageMetrics() StorageInfo {
	StorageShared := this.VirtualMachine.Summary.Storage.Unshared
	StorageCommitted := this.VirtualMachine.Summary.Storage.Committed
	StorageUnCommitted := this.VirtualMachine.Summary.Storage.Uncommitted
	NewStorageMetric := NewStorageInfo(StorageShared, StorageCommitted, StorageUnCommitted)
	return *NewStorageMetric
}
