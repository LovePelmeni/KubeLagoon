package parsers

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"log"
	"os"

	models "github.com/LovePelmeni/Infrastructure/models"
	resource_config "github.com/LovePelmeni/Infrastructure/resource_config"
	storage_config "github.com/LovePelmeni/Infrastructure/storage_config"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("../logs/Parsers.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

// Package consists of the Set of Classes, that Parses Hardware Configuration, User Specified

type DatacenterConfig struct {
	// Hardware Configuration, that is Used to Initialize Virtual Machine Server Instance

	// Datacenter Resource Info, VM will be deployed on
	Datacenter struct {
		ItemPath string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Datacenter" xml:"Datacenter"`
}

func NewHardwareConfig(Config string) (*DatacenterConfig, error) {
	var config *DatacenterConfig
	DecodedError := json.Unmarshal([]byte(Config), &config)
	return config, DecodedError
}

func (this *DatacenterConfig) GetDatacenter(Client vim25.Client) (*mo.Datacenter, error) {
	// Returns Mo Datacenter Instance, based on the Params, specified in the Config

	var MoDatacenter *mo.Datacenter
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	defer CancelFunc()

	Finder := object.NewSearchIndex(&Client)
	Datacenter, FindError := Finder.FindByInventoryPath(TimeoutContext, this.Datacenter.ItemPath)
	Collector := property.DefaultCollector(&Client)
	RetrieveError := Collector.RetrieveOne(TimeoutContext, Datacenter.Reference(), []string{"*"}, &MoDatacenter)
	if FindError != nil || RetrieveError != nil {
		return nil, errors.New("Datacenter Does Not Exist")
	} else {
		return MoDatacenter, nil
	}
}

type VirtualMachineCustomSpec struct {
	// Represents Configuration of the Virtual Machine

	Metadata struct {
		VirtualMachineId string `json:"VirtualMachineId" xml:"VirtualMachineId"`
	} `json:"Metadata" xml:"Metadata"`

	// Hardware Resourcs for the VM Configuration
	Resources struct {
		CpuNum            int32 `json:"CpuNum" xml:"CpuNum"`
		MemoryInMegabytes int64 `json:"MemoryInMegabytes" xml:"MemoryInMegabytes"`
	} `json:"Resources" xml:"Resources"`

	Disk struct {
		CapacityInKB int `json:"CapacityInKB" xml:"CapacityInKB"`
	} `json:"Disk"`
}

func NewCustomConfig(Config string) (*VirtualMachineCustomSpec, error) {
	var config VirtualMachineCustomSpec
	DecodeError := json.Unmarshal([]byte(Config), config)
	return &config, DecodeError
}

func (this *VirtualMachineCustomSpec) GetDeployConfigurations(Client vim25.Client) map[string]*types.VirtualMachineConfigSpec {
	// Parses the Custom Configuration Provided by the Client, and Converts it to the Map
	// of the Following Structure: Key - Name of the Resource and the Value is the Resource Instance

	var VirtualMachineModel models.VirtualMachine
	var VirtualMachine mo.VirtualMachine

	if Gorm := models.Database.Model(&models.VirtualMachine{}).Where("id = ?", this.Metadata.VirtualMachineId).Find(&VirtualMachineModel); Gorm.Error != nil {
		return nil
	}
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	// Receiving Mo Instance of the Virtual Machine
	VirtualMachineRef, VmError := object.NewSearchIndex(&vim25.Client{}).FindByInventoryPath(TimeoutContext, VirtualMachineModel.ItemPath)
	FindError := property.DefaultCollector(&Client).RetrieveOne(TimeoutContext,
		VirtualMachineRef.Reference(), []string{"*"}, VirtualMachine)

	if VmError != nil || FindError != nil {
		DebugLogger.Printf("Vm Not Found")
		return nil
	}

	// Setting up Disk Config
	DiskConfigSpec, DiskError := storage_config.NewVirtualMachineStorageManager().SetupStorageDisk(VirtualMachineRef.(*object.VirtualMachine),
		*storage_config.NewVirtualMachineStorage(this.Disk.CapacityInKB), &VirtualMachine.Datastore[0])

	if DiskError != nil {
		DebugLogger.Printf("Disk Configuration Failure, %s", DiskError)
		return nil
	}

	// Setting up Datastore Config
	ResourcesSpec, ResourcesError := resource_config.NewVirtualMachineResourceManager().SetupResources(
		resource_config.NewVirtualMachineResources(this.Resources.CpuNum, this.Resources.MemoryInMegabytes))

	if ResourcesError != nil {
		DebugLogger.Printf("Resource Configuration Failure, %s", ResourcesError)
		return nil
	}

	return map[string]*types.VirtualMachineConfigSpec{
		"Disk":      DiskConfigSpec,
		"Resources": ResourcesSpec,
	}
}
