package storage_config

import (
	"log"
	"os"

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
