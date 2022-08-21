package parsers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LovePelmeni/Infrastructure/suggestions"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Package consists of the Set of Classes, that Parses Hardware Configuration, User Specified

type HardwareConfig struct {
	// Hardware Configuration, that is Used to Initialize Virtual Machine Server Instance

	// Datacenter Resource Info, VM will be deployed on
	Datacenter struct {
		ItemPath string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Datacenter" xml:"Datacenter"`
}

func NewHardwareConfig(Config string) (*HardwareConfig, error) {
	var config *HardwareConfig
	DecodedError := json.Unmarshal([]byte(Config), &config)
	return config, DecodedError
}

func (this *HardwareConfig) ParseResources(Requirements suggestions.ResourceRequirements) map[string]*types.ManagedObjectReference {
	// Parses the Resource Instances for the Specific Datacenter, and Converts it Into the Map of the Name of the Resource and the value Instance

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	var Datacenter *mo.Datacenter

	// Receiving the Datacenter Reference by the Configuration, Provided by the Client

	SearchIndex := object.NewSearchIndex(&vim25.Client{})
	DatacenterRef, FindError := SearchIndex.FindByInventoryPath(TimeoutContext, this.Datacenter.ItemPath)

	// Receiving the Datacenter Instance by the Datacenter reference
	Collector := property.DefaultCollector(&vim25.Client{})
	RetrieveError := Collector.RetrieveOne(TimeoutContext, DatacenterRef.Reference(), []string{"*"}, &Datacenter)
	ResourceManager := suggestions.NewDataCenterSuggestManager(vim25.Client{})
	Resources := ResourceManager.GetDatacenterResources(Datacenter, Requirements)

	switch {
	case RetrieveError != nil || FindError != nil:
		return map[string]*types.ManagedObjectReference{}
	default:
		return Resources
	}
}

type VirtualMachineCustomSpec struct {
	// Represents Configuration of the Virtual Machine

	Metadata struct {
		VirtualMachineName string `json:"VirtualMachineName" xml:"VirtualMachineName"`
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

func (this *VirtualMachineCustomSpec) ParseResources() map[string]*types.VirtualMachineConfigSpec {
	// Parses the Custom Configuration Provided by the Client, and Converts it to the Map
	// of the Following Structure: Key - Name of the Resource and the Value is the Resource Instance

}
