package resource_config

import (
	"log"
	"os"

	"github.com/vmware/govmomi/vim25/types"
)

var (
	DebugLogger *log.Logger
	ErrorLogger *log.Logger
	InfoLogger  *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("ResourceConfig.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
	SetupResources(Resources VirtualMachineResources) (*types.VirtualMachineConfigSpec, error)
}

type VirtualMachineResourceManager struct {
	VirtualMachineResourceManagerInterface
}

func NewVirtualMachineResourceManager() VirtualMachineResourceManager {
	return VirtualMachineResourceManager{}
}

func SetupResources(Resources VirtualMachineResources) (*types.VirtualMachineConfigSpec, error) {

	ResourceSpecification := types.VirtualMachineConfigSpec{
		NumCPUs:             Resources.CpuNum,
		NumCoresPerSocket:   Resources.CpuNum / 2,
		MemoryMB:            1024 * Resources.MemoryInMegabytes,
		CpuHotAddEnabled:    types.NewBool(true),
		MemoryHotAddEnabled: types.NewBool(true),
	}
	return &ResourceSpecification, nil
}
