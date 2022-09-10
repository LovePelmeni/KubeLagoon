package resource_config

import (
	"os"

	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("ResourceConfigLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
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
