package parsers

import (
	"context"
	"encoding/json"

	"errors"
	"time"

	"log"
	"os"

	"github.com/LovePelmeni/Infrastructure/host_system"
	"github.com/LovePelmeni/Infrastructure/models"

	"github.com/LovePelmeni/Infrastructure/network"
	resource_config "github.com/LovePelmeni/Infrastructure/resource_config"
	"github.com/LovePelmeni/Infrastructure/ssh_config"

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
	LogFile, Error := os.OpenFile("Parsers.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
		VmOwnerId        string `json:"VmOwnerId" xml:"VmOwnerId"`
	} `json:"Metadata" xml:"Metadata"`

	LoadBalancer struct {
		Ssl string `json:"Ssl" xml:"Ssl"`
	} `json:"LoadBalancer" xml:"LoadBalancer"`

	HostSystem struct {
		Type             string `json:"Type"` // OS Distribution Type Like: Linux, Windows etc....
		DistributionName string `json:"DistributionName"`
		Bit              int64  `json:"Bit;omitempty"`
		Version          string `json:"Version"`
	} `json:"HostSystem"`

	Ssh struct {
		Type string `json:"Type"`
	}

	Network struct {
		IP       string `json:"IP,omitempty"`
		Netmask  string `json:"Netmask,omitempty"`
		Hostname string `json:"Hostname,omitempty"`
		Gateway  string `json:"Gateway,omitempty"`
		Enablev6 bool   `json:"Enablev6,omitempty"`
		Enablev4 bool   `json:"Enablev4,omitempty"`
	} `json:"Network" xml:"Network"`

	// Extra Tools, that is going to be Installed on the VM automatically
	// Things Like Docker, Docker-Compose, VirtualBox or Podman etc....

	ExtraTools struct {
		Tools []string `json:"Tools" xml:"Tools"` // Names of the Tools
	} `json:"ExtraTools;omitempty" xml:"ExtraTools"`

	// Hardware Resourcs for the VM Configuration
	Resources struct {
		CpuNum            int32 `json:"CpuNum" xml:"CpuNum"`
		MemoryInMegabytes int64 `json:"MemoryInMegabytes" xml:"MemoryInMegabytes"`
		MaxMemoryUsage    int64 `json:"MaxMemoryUsage,omitempty;" xml:"MaxMemoryUsage"`
		MaxCpuUsage       int64 `json:"MaxCpuUsage,omitempty;" xml:"MaxCpuUsage"`
	} `json:"Resources" xml:"Resources"`

	Disk struct {
		CapacityInKB int `json:"CapacityInKB" xml:"CapacityInKB"`
	} `json:"Disk" xml:"Disk"`
}

func NewCustomConfig(Config string) (*VirtualMachineCustomSpec, error) {
	var config VirtualMachineCustomSpec
	DecodeError := json.Unmarshal([]byte(Config), &config)
	return &config, DecodeError
}

func (this *VirtualMachineCustomSpec) ToJson() []byte {
	EncodedConfiguration, _ := json.Marshal(this)
	return EncodedConfiguration
}

func (this *VirtualMachineCustomSpec) ToStruct(SerializedConfiguration []byte) VirtualMachineCustomSpec {
	json.Unmarshal(SerializedConfiguration, this)
	return *this
}

func (this *VirtualMachineCustomSpec) GetHostSystemConfig(Client vim25.Client) (types.VirtualMachineGuestSummary, types.CustomizationSpec, error) {

	// Converting JSON Host System Configuration, Provided By Customer, to the Configuration Instance

	HostSystemManager := host_system.NewVirtualMachineHostSystemManager()
	HostSystemCredentials := host_system.NewHostSystemCredentials(this.HostSystem.DistributionName, this.HostSystem.Version, this.HostSystem.Bit)

	HostSystemConfiguration, HostSystemCustomizationConfig, SetupError := HostSystemManager.SetupHostSystem(*HostSystemCredentials)
	return *HostSystemConfiguration, *HostSystemCustomizationConfig, SetupError
}

func (this *VirtualMachineCustomSpec) GetResourceConfig(Client vim25.Client) (types.VirtualMachineConfigSpec, error) {

	// Converting JSON Resource Configuration, Provided By Customer, to the Configuration Instance

	ResourceCredentials := resource_config.NewVirtualMachineResources(this.Resources.CpuNum, this.Resources.MemoryInMegabytes)
	ResourceManager := resource_config.NewVirtualMachineResourceManager()

	ResourceConfiguration, ResourceError := ResourceManager.SetupResources(ResourceCredentials)
	return *ResourceConfiguration, ResourceError
}

func (this *VirtualMachineCustomSpec) GetDiskStorageConfig(Client vim25.Client) (*types.VirtualDeviceConfigSpec, error) {

	// Converting JSON Disk Storage Configuration, Provided By Customer, to te Configuration Instance

	// Receiving Virtual Machine by the Metadata, Provided in the Configuration...
	VirtualMachine, FindError := func() (*object.VirtualMachine, error) {
		var Vm models.VirtualMachine

		TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
		defer CancelFunc()

		Gorm := models.Database.Model(&models.VirtualMachine{}).Where("id = ? AND owner_id = ?",
			this.Metadata.VirtualMachineId, this.Metadata.VmOwnerId).Find(&Vm)
		if Gorm.Error != nil {
			return nil, Gorm.Error
		}

		VirtualMachine, FindError := object.NewSearchIndex(&Client).FindByInventoryPath(TimeoutContext, Vm.ItemPath)
		return VirtualMachine.(*object.VirtualMachine), FindError
	}()

	if FindError != nil {
		return nil, FindError
	}

	Datastore := object.NewDatastore(&Client, object.NewReference(&Client, VirtualMachine.Reference()).(*mo.VirtualMachine).Datastore[0])
	DiskDeviceStorageCredentials := storage_config.NewVirtualMachineStorage(this.Disk.CapacityInKB)
	DiskDeviceManager := storage_config.NewVirtualMachineStorageManager()

	Configuration, SetupError := DiskDeviceManager.SetupStorageDisk(*DiskDeviceStorageCredentials, *Datastore)
	return Configuration, SetupError
}

func (this *VirtualMachineCustomSpec) GetNetworkConfig(Client vim25.Client) (*types.CustomizationSpec, error) {
	// Returns Virtual Machine Network Configuration for the Virtual Machine
	IPCredentials := network.NewVirtualMachineIPAddress(this.Network.IP, this.Network.Netmask, this.Network.Gateway, this.Network.Hostname)
	NewNetworkManager := network.NewVirtualMachinePublicNetworkManager()
	NetworkConfig, SetupError := NewNetworkManager.SetupPublicNetwork(*IPCredentials)
	return NetworkConfig, SetupError
}

func (this *VirtualMachineCustomSpec) GetExtraToolsConfig(Client vim25.Client) ([]string, error) {
	// Returns Installation Tools
	return this.ExtraTools.Tools, nil
}

func (this *VirtualMachineCustomSpec) ApplySshConfig(Client vim25.Client, VirtualMachine *object.VirtualMachine) (interface{}, error) {
	// Returns SSH Support Configuration for the Virtual Machine, based on the Config
	// That Customer Has Specified
	switch {
	case this.Ssh.Type == models.TypeByRootCredentials:
		newCertificateManager := ssh_config.NewVirtualMachineSshCertificateManager(Client, VirtualMachine)
		PublicKey, SslCertificateError := newCertificateManager.GenerateSshKeys()
		return PublicKey, SslCertificateError

	case this.Ssh.Type == models.TypeByRootCertificate:
		newRootCredentialsManager := ssh_config.NewVirtualMachineSshRootCredentialsManager(Client, VirtualMachine)
		RootCredentials, SslRootError := newRootCredentialsManager.GetSshRootCredentials()
		return RootCredentials, SslRootError
	default:
		return nil, errors.New("SSH Disabled")
	}
}
