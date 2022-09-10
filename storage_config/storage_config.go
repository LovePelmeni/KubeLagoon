package storage_config

import (
	"os"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {
	// Initializing ZAP Logger
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("ResourcesLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
}

type VirtualMachineStorage struct {
	// Structure, represents Data Storage Capacity parameters, that will be eventually
	// Applied to the Virtual Machine
	DiskCapacityKB int64
}

func NewVirtualMachineStorage(CapacityInKB int) *VirtualMachineStorage {
	return &VirtualMachineStorage{
		DiskCapacityKB: int64(CapacityInKB),
	}
}

type VirtualMachineStorageManagerInterface interface {
	// Interface, represents Manager Class, for handling Storage Resources of the Virtual Machine
	SetupStorageDisk(VirtualMachine *object.VirtualMachine, Storage VirtualMachineStorage) (*types.VirtualMachineConfigSpec, error)
}

type VirtualMachineStorageManager struct {
	// Manager Class, for handling Storage Resources of the Virtual Machine
	VirtualMachineStorageManagerInterface
}

func NewVirtualMachineStorageManager() *VirtualMachineStorageManager {
	return &VirtualMachineStorageManager{}
}

func (this *VirtualMachineStorageManager) SetupStorageDisk(

	StorageCredentials VirtualMachineStorage,
	DataStore object.Datastore,

) (*types.VirtualDeviceConfigSpec, error) {

	// Initializing New Virtual Disk

	ReferencedDatastore := DataStore.Reference()
	DeviceDisk := types.VirtualDisk{

		CapacityInKB: StorageCredentials.DiskCapacityKB,

		VirtualDevice: types.VirtualDevice{
			Backing: &types.VirtualDiskFlatVer2BackingInfo{

				DiskMode:        string(types.VirtualDiskModePersistent),
				ThinProvisioned: types.NewBool(true),
				VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
					Datastore: &ReferencedDatastore,
				},
			},
		},
	}
	DeviceSpec := &types.VirtualDeviceConfigSpec{
		Operation:     types.VirtualDeviceConfigSpecOperationAdd,
		FileOperation: types.VirtualDeviceConfigSpecFileOperationCreate,
		Device:        &DeviceDisk,
	}
	return DeviceSpec, nil
}
