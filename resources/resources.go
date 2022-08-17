package resources

import (
	"context"
	"time"

	"log"
	"os"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	DebugLogger *log.Logger
	ErrorLogger *log.Logger
	InfoLogger  *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("Resources.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

type VirtualMachineResources struct {
	// Structure Represents Virtual Machine Resources
	// CPU and Memory (in Megabytes), that is Going to be Allocated to the Virtual Machine

	CpuNum            int32
	MemoryInMegabytes int64
}

func NewVirtualMachineResources(CpuNum int32, MemoryInMegabytes int64) VirtualMachineResources {
	return VirtualMachineResources{
		CpuNum:            CpuNum,
		MemoryInMegabytes: MemoryInMegabytes,
	}
}

type VirtualMachineResourceManagerInterface interface {
	// Interface, represents Class Manager, that Setting up CPU and Memory Settings to the Virtual Machine
	// * Setting up Following Resources to the Virtual Machine (
	//  CPU, Memory in Megabytes
	// )
	SetupResources(Resources VirtualMachineResources) (VirtualMachineResources, error)
}

type VirtualMachineResourceManager struct{}

func NewVirtualMachineResourceManager() VirtualMachineResourceManager {
	return VirtualMachineResourceManager{}
}

func SetupResources(VirtualMachine *object.VirtualMachine, Resources VirtualMachineResources) (*VirtualMachineResources, error) {

	CustomizedSpecification := types.VirtualMachineConfigSpec{
		NumCPUs:             Resources.CpuNum, // Setting up Numbers of CPU's
		NumCoresPerSocket:   Resources.CpuNum / 2,
		MemoryMB:            Resources.MemoryInMegabytes * 1024, // setting up Memory In Megabytes
		CpuHotAddEnabled:    types.NewBool(true),
		MemoryHotAddEnabled: types.NewBool(true),
	}
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	NewTask, CustomizeError := VirtualMachine.Reconfigure(
		TimeoutContext, CustomizedSpecification)

	AppliedError := NewTask.Wait(TimeoutContext)

	switch {
	case CustomizeError != nil || AppliedError != nil:
		ErrorLogger.Printf("Failed to Apply Custom Resources for CPUs and Memory, Errors: [%s, %s]",
			CustomizeError, AppliedError)
		return nil, exceptions.ResourcesSetupFailure()

	case CustomizeError == nil && AppliedError == nil:
		DebugLogger.Printf("Customized CPU's and Memory has been Applied Successfully.")
		return &Resources, nil
	default:
		return &Resources, nil
	}
}
