package parsers

import (
	"encoding/json"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/LovePelmeni/Infrastructure/suggestions"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"
)

// Package consists of the Set of Classes, that Parses Hardware Configuration, User Specified

type HardwareConfig struct {

	// Hardware Configuration, that is Used to Initialize Virtual Machine Server Instance

	// Network Resource info, that VM will be Connected to
	Network struct {
		ItemPath string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Network" xml:"Network"`

	// Datacenter Resource Info, VM will be deployed on
	Datacenter struct {
		ItemPath string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Datacenter" xml:"Datacenter"`

	// Datastore Resource Info, VM will be using for storing Data
	DataStore struct {
		ItemPath string `json:"ItemPath" xml:"ItemPath"`
	} `json:"DataStore" xml:"Datastore"`

	// Place, where Physical Resources is going to be Picked Up From, it can be HostMachine or Cluster
	ResourcePool struct {
		ItemPath string `json:"ItemPath" xml:"ItemPath"`
	} `json:"ResourcePool" xml:"ResourcePool"`

	// Forder Resource Info, where the Info about VM is going to be Stored.
	Folder struct {
		ItemPath string `json:"ItemPath" xml:"ItemPath"`
	} `json:"Folder" xml:"Folder"`
}

func NewHardwareConfig(Config string) (*HardwareConfig, error) {
	var config *HardwareConfig
	DecodedError := json.Unmarshal([]byte(Config), &config)
	return config, DecodedError
}

func (this *HardwareConfig) GetResources(Client vim25.Client) (map[string]object.Reference, error) {

	var Resources map[string]object.Reference
	ResourceManager := suggestions.NewResourceSuggestManager(Client)
	for ResourceName, Instance := range map[string]struct{ ItemPath string }{
		"Network":      struct{ ItemPath string }(this.Network),
		"Datastore":    struct{ ItemPath string }(this.DataStore),
		"Folder":       struct{ ItemPath string }(this.Folder),
		"ResourcePool": struct{ ItemPath string }(this.ResourcePool),
	} {
		Resource, Error := ResourceManager.GetResource(Instance.ItemPath)
		if Error != nil {
			return nil, exceptions.ItemDoesNotExist()
		} else {
			Resources[ResourceName] = Resource
		}
	}
	return Resources, nil
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
