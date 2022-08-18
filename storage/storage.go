package storage

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
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("Storage.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

type VirtualMachineStorage struct {
	// Structure, represents Data Storage Capacity parameters, that will be eventually
	// Applied to the Virtual Machine
	DiskCapacityKB int64
}

func NewVirtualMachineStorage() *VirtualMachineStorage {
	return &VirtualMachineStorage{}
}

type VirtualMachineStorageManagerInterface interface {
	// Interface, represents Manager Class, for handling Storage Resources of the Virtual Machine
	SetStorage(Storage VirtualMachineStorage) (VirtualMachineStorage, error)
}

type VirtualMachineStorageManager struct {
	// Manager Class, for handling Storage Resources of the Virtual Machine
	VirtualMachineStorageManagerInterface
}

func NewVirtualMachineStorageManager() *VirtualMachineStorageManager {
	return &VirtualMachineStorageManager{}
}

func (this *VirtualMachineStorageManager) SetupStorageDisk(

	VirtualMachine *object.VirtualMachine,
	StorageCredentials VirtualMachineStorage,
	DataStore *types.ManagedObjectReference,

) (*types.VirtualDeviceConfigSpec, error) {

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	// Initializing New Device
	Devices, DeviceError := VirtualMachine.Device(TimeoutContext)
	DiskController, ControllerError := Devices.FindDiskController("scsi")

	if DeviceError != nil || ControllerError != nil {
		return nil, exceptions.StorageSetupFailure()
	}
	// Initializing New Virtual Disk

	DeviceDisk := types.VirtualDisk{

		CapacityInKB: StorageCredentials.DiskCapacityKB,

		VirtualDevice: types.VirtualDevice{
			Backing: &types.VirtualDiskFlatVer2BackingInfo{

				DiskMode:        string(types.VirtualDiskModePersistent),
				ThinProvisioned: types.NewBool(true),
				VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
					Datastore: DataStore,
				},
			},
		},
	}
	// Assigning New Device Controller
	Devices.AssignController(&DeviceDisk, DiskController)
	DeviceSpec := &types.VirtualDeviceConfigSpec{
		Operation:     types.VirtualDeviceConfigSpecOperationAdd,
		FileOperation: types.VirtualDeviceConfigSpecFileOperationCreate,
		Device:        &DeviceDisk,
	}

	return DeviceSpec, nil
}
