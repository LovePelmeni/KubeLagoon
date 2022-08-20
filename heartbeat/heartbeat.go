package heartbeat

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/progress"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("heartbeat.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if Error != nil {
		panic(Error)
	}
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Package offers a number of different Classes, that is used for Perform Healthchecks
// Onto the Virtual Machine

type VirtualMachineHeartbeatMetric struct {
	CpuHotPercent int   `json:"CpuHotPersent"`
	MemoryInUse   int32 `json:"MemoryInUse"`
}

func NewVirtualMachineHeartBeatMetric(CpuHot int, MemoryInUse int32) *VirtualMachineHeartbeatMetric {
	return &VirtualMachineHeartbeatMetric{
		CpuHotPercent: CpuHot,
		MemoryInUse:   MemoryInUse,
	}
}

type HealthStateReportManager struct {
	// Health State Reporter Manager, that reports Health State of the Virtual Server
}

func (this *HealthStateReportManager) Sink() chan<- progress.Report {
}

type VirtualMachineHeartBeatManager struct{}

func NewVirtualMachineHeartBeatManager() *VirtualMachineHeartBeatManager {
	return &VirtualMachineHeartBeatManager{}
}

func (this *VirtualMachineHeartBeatManager) EnableMetrics(VirtualMachine *object.VirtualMachine) error {

	// Method, enables HealthCheck Configuration on the Virtual Server, so the Other Services, can Parse it Via API

	VirtualMachineConfig := types.VirtualMachineConfigSpec{
		VPMCEnabled:         types.NewBool(true),
		CpuHotAddEnabled:    types.NewBool(true),
		MemoryHotAddEnabled: types.NewBool(true),
	}
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	// Checking The Power State of the VM before, applying any changes (ITS IMPORTANT)
	Running, StatusError := VirtualMachine.PowerState(TimeoutContext)

	if strings.Contains(string(Running), "Running") {
		ShutdownError := VirtualMachine.ShutdownGuest(TimeoutContext)
		if ShutdownError != nil {
			ErrorLogger.Printf("Failed to Shutdown Virtual Machine with Unique Name: %s",
				VirtualMachine.Reference().Value)
			return ShutdownError
		}
	}
	if StatusError != nil {
		ErrorLogger.Printf("Failed to Parse VM State, Error: %s")
		return StatusError
	}

	// Reconfiguring Virtual Machine, to add Healthcheck metrics Configuration
	ApplyHealthMetricsTask, _ := VirtualMachine.Reconfigure(TimeoutContext, VirtualMachineConfig)
	Wait := ApplyHealthMetricsTask.Wait(TimeoutContext)

	switch {
	case len(Wait.Error()) != 0:
		ErrorLogger.Printf("Failed to Apply ")
		return Wait
	case len(Wait.Error()) == 0:
		DebugLogger.Printf("HealthChecks Metrics has been Enabled on Virtual Machine with UniqueName: %s",
			VirtualMachine.Reference().Value)
		return nil
	default:
		return nil
	}
}

func (this *VirtualMachineHeartBeatManager) GetCpuMetrics(IPUrl string, Network string) {
}

func (this *VirtualMachineHeartBeatManager) GetMemoryUsageMetrics() {
}

func (this *VirtualMachineHeartBeatManager) GetMetrics() {
}
