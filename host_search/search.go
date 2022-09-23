package search

import "encoding/json"

// Package that searches for the appropriate host machine, that is going to have at least over 10 % of the resources
// so the Customer will be uaranteed, that his Virtual Machine Server, is not gonna be out of resources

type HostMachineRequirementsInterface interface {
	// Interface reprensents Base requirements for the Host Machine
}

type HostMachineRequirements struct {
	HostMachineRequirementsInterface
	Resources struct {
		Cpus                   int32 `json:"Cpus" xml:"Cpus"`
		TotalMemoryInMegabytes int32 `json:"TotalMemory" xml:"TotalMemory"`
		DiskStorageInMegabytes int32 `json:"TotalDiskStorage" xml:"TotalDiskStorage"`
	}
}

func NewHostMachineRequirements() *HostMachineRequirements {
	return &HostMachineRequirements{}
}

type HostMachineInterface interface {
	// Base Interface, representing Host Machine
}

type HostMachine struct {
	HostMachineInterface
	// Host Machine
	HostMachineIP        string `json:"HostMachineIP"`
	HostMachineResources struct {
		Cpus                   int32 `json:"Cpus" xml:"Cpus"`
		TotalMemoryInMegabytes int32 `json:"TotalMemoryInMegabytes" xml:"TotalMemoryInMegabytes"`
	} `json:"HostMachineResources" xml:"HostMachineResources"`
	Client struct {
		SourceIpAddress string `json:"SourceIpAddress"`
		SourceUser      string `json:"SourceUser"`
		SourcePassword  string `json:"SourcePassword"`
	} `json:"Client" xml:"Client"`
}

func NewHostMachine(SerializedHostMachineString []byte) (HostMachine, error) {
	var hostMachine HostMachine
	DecodeError := json.Unmarshal(SerializedHostMachineString, &hostMachine)
	return hostMachine, DecodeError
}

type HostMachineSearcherInterface interface {
	// Searches for the
	GetAllHostMachines() []HostMachine
	SearchHostMachine(HostMachines []HostMachine) HostMachine
}

type HostMachineSearcher struct {
	HostMachineSearcherInterface
}

func NewHostMachineSearcher() *HostMachineSearcher {
	return &HostMachineSearcher{}
}
func (this *HostMachineSearcher) GetAllHostMachines() ([]HostMachine, error) {
	// Returns an array of the All Available Host Machines, with their credentials
}

func (this *HostMachineSearcher) SearchHostMachine(HostMachines []HostMachine, Requirements HostMachineRequirementsInterface) HostMachine {
	// Returns an appropriate Host Machine, based on the Resources, that is available within it and Requirements
}
