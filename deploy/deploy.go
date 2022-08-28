package deploy

import (
	"context"
	"errors"
	"log"
	"sync"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/LovePelmeni/Infrastructure/parsers"
	"github.com/LovePelmeni/Infrastructure/ssh_config"

	"github.com/LovePelmeni/Infrastructure/models"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/simulator/esx"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/library"
	"github.com/vmware/govmomi/vapi/rest"

	"github.com/vmware/govmomi/vapi/vcenter"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("Deploy.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

type ResourceKeys struct {
	// Credentials
	// * For the Storage: `Storage Key` - represents Acccess Key to the Storage, where All the Data will Be Allocated.
	// * For the Network: `Network Key` - represents Access Key to the Network, the Application will be Attached to.
	NetworkKey string `json:"NetworkKey" xml:"NetworkKey"`
	StorageKey string `json:"StorageKey" xml:"StorageKey"`
}

func NewDeployResourceKeys(NetworkKey string, StorageKey string) *ResourceKeys {
	return &ResourceKeys{
		NetworkKey: NetworkKey,
		StorageKey: StorageKey,
	}
}

type VmInfo struct {
	// Response Configuration after the Config has been Applied Successfully
	IPAddress        string               `json:"IPAddress"`
	SshPublicKey     ssh_config.PublicKey `json:"PublicKey"`
	NetworkIPAddress string               `json:"NetworkIPAddress"`
}

type VirtualMachineResourceKeyManager struct {

	// Class, that is responsible for Obtaining Credentials, to the Storage/Network
	// To Get the Permissions to use that Resources.

	Client rest.Client
}

func NewVirtualMachineResourceKeyManager() *VirtualMachineResourceKeyManager {
	return &VirtualMachineResourceKeyManager{}
}

func (this *VirtualMachineResourceKeyManager) GetLibraryItem(Context context.Context) (*library.Item, error) {
	// Returning Library Item
	const (
		libName         = ""
		libItemName     = ""
		libraryItemType = "ovf"
	)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	m := library.NewManager(&this.Client)

	libraries, Error := m.FindLibrary(TimeoutContext, library.Find{Name: libName})
	if Error != nil {
		return nil, Error
	}

	if len(libraries) == 0 {
		return nil, errors.New("No Libraries Found")
	}

	if len(libraries) > 1 {
		return nil, errors.New("Go multiple Libraries")
	}

	//  ovf   ovf
	items, ParseError := m.FindLibraryItems(TimeoutContext, library.FindItem{Name: libItemName,
		Type: libraryItemType, LibraryID: libraries[0]})

	if ParseError != nil {
		return nil, ParseError
	}

	if len(items) == 0 {
		return nil, errors.New("No Items has been Found")
	}

	if len(items) > 1 {
		return nil, errors.New("Got Multiple Items")
	}

	item, GetError := m.GetLibraryItem(TimeoutContext, items[0])
	if GetError != nil {
		return nil, GetError
	}
	return item, nil
}

func (this *VirtualMachineResourceKeyManager) GetResourceKeys(
	Resource *object.ResourcePool, Folder *object.Folder) (*ResourceKeys, error) {

	// Returns Storage and Resource Key in order to get Usage Permissions
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*20)
	defer CancelFunc()

	LibraryItem, LibError := this.GetLibraryItem(TimeoutContext)
	DatacenterManager := vcenter.NewManager(&this.Client)

	FilterRequest := vcenter.FilterRequest{Target: vcenter.Target{
		ResourcePoolID: Resource.Reference().Value,
		FolderID:       Folder.Reference().Value,
	}}

	Filter, FilterError := DatacenterManager.FilterLibraryItem(
		TimeoutContext, LibraryItem.ID, FilterRequest)

	switch {

	case LibError != nil || FilterError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case LibError == nil || FilterError == nil:
		ResourceKeys := NewDeployResourceKeys(
			Filter.Networks[0], Filter.StorageGroups[0])
		return ResourceKeys, nil

	default:
		return nil, exceptions.ItemDoesNotExist()
	}
}

type VirtualMachineManager struct {
	Reconfig sync.RWMutex
	// Class, that Is Taking care of the Virtual Machine Deployment Process
	// It is responsible for Deploying/ Starting / Stopping / Updating Virtual Machines
	// Owned by Customers
	// Provides Following Methods in order to Fullfill the Needs and make the Process comfortable and easier
	VimClient vim25.Client
}

func NewVirtualMachineManager(Client vim25.Client) *VirtualMachineManager {
	return &VirtualMachineManager{
		VimClient: Client,
	}
}

func (this *VirtualMachineManager) GetVirtualMachine(VmId string, CustomerId string) (*object.VirtualMachine, error) {

	// Method Retunrs Prepared Virtual Machine Instance, (That Already Exists, and has been created by Customer)

	// Initializing Timeout Context
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	// Receiving Virtual Machine Configuration Instance from the Database

	var VirtualMachineObj models.VirtualMachine

	VirtualMachineGormRefError := models.Database.Model(
		&models.VirtualMachine{}).Where(
		"owner_id = ? AND id = ?",
		CustomerId, VmId).Find(&VirtualMachineObj)

	if VirtualMachineGormRefError != nil {
		ErrorLogger.Printf("Failed to Find Virtual Machine in Database with ID: %s and Owner ID: %s", VmId, CustomerId)
		return nil, exceptions.ItemDoesNotExist()
	}

	// Receiving Prepared Virtual Machine Instance by using API

	FinderIndex := object.NewSearchIndex(&this.VimClient)
	VirtualRef, FindError := FinderIndex.FindByInventoryPath(TimeoutContext, VirtualMachineObj.ItemPath)

	switch {
	case FindError != nil:
		ErrorLogger.Printf("Failed to Find Virtual Machine with ID: %s of Customer: %s", VmId, CustomerId)
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return VirtualRef.(*object.VirtualMachine), nil

	default:
		return VirtualRef.(*object.VirtualMachine), nil
	}
}

func (this *VirtualMachineManager) InitializeNewVirtualMachine(
	VimClient vim25.Client,
	VirtualMachineName string,
	DataStore *object.Datastore, // Name, Customer Decided to set up for this Virtual Server
	DatacenterNetwork *object.Network,
	DatacenterClusterComputeResource *object.ClusterComputeResource,
	DatacenterFolder *object.Folder,
) (*object.VirtualMachine, error) {
	// Initializes Virtual Machine Configuration (That does not exist yet)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	// Receiving Cluster's Resource Pool
	ClusterResourcePool, ResourcePoolError := DatacenterClusterComputeResource.ResourcePool(TimeoutContext)

	// If Failed to Get Clusters Resource Pool, returning Exception
	if ResourcePoolError != nil {
		ErrorLogger.Printf("Failed to Get Cluster Resource Pool, Error: %s", ResourcePoolError)
		return *new(*object.VirtualMachine), errors.New("Failed to Get Cluster Resource Pool")
	}

	// Initializing Virtual Machine Resource Key Manager, that Is Going to Obtain Necessasy Keys
	// In order to Get Access to Resource Pools

	ResourceAllocationManager := NewVirtualMachineResourceKeyManager()
	ResourceCredentials, ResourceKeysError := ResourceAllocationManager.GetResourceKeys(
		ClusterResourcePool, DatacenterFolder)

	switch {

	// If Resources Keys has been Received, Going through the Next Steps
	case ResourceKeysError == nil && len(ResourceCredentials.NetworkKey) != 0 && len(ResourceCredentials.StorageKey) != 0:

		Deployment := vcenter.Deploy{
			DeploymentSpec: vcenter.DeploymentSpec{

				Name:               VirtualMachineName,
				DefaultDatastoreID: DataStore.Reference().Value,
				AcceptAllEULA:      true,
				NetworkMappings: []vcenter.NetworkMapping{{
					Key:   ResourceCredentials.NetworkKey,
					Value: DatacenterNetwork.Reference().Value,
				}},

				StorageMappings: []vcenter.StorageMapping{
					{
						Key: ResourceCredentials.StorageKey,
						Value: vcenter.StorageGroupMapping{
							Type:         "DATASTORE",
							DatastoreID:  DataStore.Reference().Value,
							Provisioning: "thin",
						},
					},
				},
				StorageProvisioning: "thin",
			},
			Target: vcenter.Target{
				ResourcePoolID: ClusterResourcePool.Reference().Value,
				FolderID:       DatacenterFolder.Reference().Value,
			},
		}

		DeployTimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
		defer CancelFunc()

		Item, ItemError := ResourceAllocationManager.GetLibraryItem(TimeoutContext)

		if ItemError != nil {
			DebugLogger.Printf("ItemError: %s", ItemError)
			return nil, exceptions.ItemDoesNotExist()
		}

		VirtualMachineInstanceReference, DeployError := vcenter.NewManager(rest.NewClient(&VimClient)).DeployLibraryItem(DeployTimeoutContext, Item.ID, Deployment)

		switch {
		case DeployError != nil:
			ErrorLogger.Printf("Failed to Deploy Virtual Machine From Library, Error: %s", DeployError)
			return nil, exceptions.DeployFromLibraryFailure()

		case DeployError == nil:
			DebugLogger.Printf("Virtual Machine Has been Deployed Successfully from Library, Obtaining Reference Resource")

			newFinder := find.NewFinder(&VimClient)
			VmRef, FindError := newFinder.ObjectReference(DeployTimeoutContext, *VirtualMachineInstanceReference)

			if FindError != nil {
				ErrorLogger.Printf("VM Does Not Exist or Could Not Be Found After Deploy, Error: %s", FindError)
				return nil, exceptions.DeployFromLibraryFailure()
			} else {
				DebugLogger.Printf("New Virtual Machine has been Initialized Successfully.")
				return VmRef.(*object.VirtualMachine), nil
			}
		}

	case ResourceKeysError != nil:
		DebugLogger.Printf("No Resources Is Available. to Deploy New Virtual Machine")
		return nil, exceptions.NoResourceAvailable()

	default:
		DebugLogger.Printf("Unknown Deploy Errors: [%s]", ResourceKeysError)
		return nil, exceptions.VMDeployFailure()
	}
	return nil, exceptions.VMDeployFailure()
}

func (this *VirtualMachineManager) ApplyConfiguration(VirtualMachine *object.VirtualMachine, Configuration parsers.VirtualMachineCustomSpec) (*VmInfo, error) {

	// Applies Custom Configuration: Num's of CPU's, Memory etc... onto the Initialized Virtual Machine

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)

	// Receiving Virtual Machine Configurations to Apply

	// Receiving OS HostSystem Config
	HostSystemConfig, HostSystemCustomizationConfig, HostSystemError := Configuration.GetHostSystemConfig(this.VimClient) // HostSystem Configuration for the Vm
	if HostSystemError != nil {
		return nil, HostSystemError
	}
	// Getting Resource Storage Config
	ResourceConfig, ResourceError := Configuration.GetResourceConfig(this.VimClient) // Resource (CPU, Memory) Configuration for the VM
	if ResourceError != nil {
		return nil, ResourceError
	}

	// Getting Disk Storage Config
	DiskStorageConfig, DiskError := Configuration.GetDiskStorageConfig(this.VimClient) // Disk Storage Configuration for the VM
	if DiskError != nil {
		return nil, DiskError
	}

	// Getting Network Config
	NetworkConfig, NetworkError := Configuration.GetNetworkConfig(this.VimClient)
	if NetworkError != nil {
		return nil, NetworkError
	}

	// Getting Extra Tools Configuration, that is going to be Installed on the VM
	_, ExtraToolsError := Configuration.GetExtraToolsConfig(this.VimClient)
	if ExtraToolsError != nil {
		ErrorLogger.Printf("Failed to Obtain Extra Tools that Should be Installed on the VM, Error: %s",
			ExtraToolsError)
	}

	// Retrieving Mo Entity of the Virtual Machine
	var vm mo.VirtualMachine
	Collector := property.DefaultCollector(&this.VimClient)
	RetrieveError := Collector.RetrieveOne(TimeoutContext, VirtualMachine.Reference(), []string{"*"}, &vm)
	if RetrieveError != nil {
		return nil, errors.New("VM Server not found")
	}

	rspec := types.DefaultResourceConfigSpec()
	vm.Guest = &types.GuestInfo{GuestId: HostSystemConfig.GuestId}
	vm.Config = &types.VirtualMachineConfigInfo{
		ExtraConfig:        []types.BaseOptionValue{&types.OptionValue{Key: "govcsim", Value: "TRUE"}},
		MemoryAllocation:   &rspec.MemoryAllocation,
		CpuAllocation:      &rspec.CpuAllocation,
		LatencySensitivity: &types.LatencySensitivity{Level: types.LatencySensitivitySensitivityLevelNormal},
		BootOptions:        &types.VirtualMachineBootOptions{},
		CreateDate:         types.NewTime(time.Now()),
		InitialOverhead: &types.VirtualMachineConfigInfoOverheadInfo{
			InitialMemoryReservation: 100, // In Megabytes
			InitialSwapReservation:   100, // In Megabytes
		},
	}
	vm.Layout = &types.VirtualMachineFileLayout{}
	vm.LayoutEx = &types.VirtualMachineFileLayoutEx{
		Timestamp: time.Now(),
	}
	vm.Snapshot = nil // intentionally set to nil until a snapshot is created
	vm.Storage = &types.VirtualMachineStorageInfo{
		Timestamp: time.Now(),
	}
	vm.Summary.Guest = &HostSystemConfig
	vm.Summary.Vm = &vm.Self
	vm.Summary.Storage = &types.VirtualMachineStorageSummary{
		Timestamp: time.Now(),
	}


	// Applying Max CPU/Memory Usage to the Virtual Machine Server 


	if Configuration.Resources.MaxCpuUsage != 0 {
		vm.Summary.Runtime.MaxCpuUsage = int32(Configuration.Resources.MaxCpuUsage)
	}

	if Configuration.Resources.MaxMemoryUsage != 0 {
		vm.Summary.Runtime.MaxMemoryUsage = int32(Configuration.Resources.MaxMemoryUsage)
	}

	// Initializing New Devices

	Devices, DeviceError := VirtualMachine.Device(TimeoutContext)

	if DeviceError != nil {
		ErrorLogger.Printf(DeviceError.Error())
		return nil, errors.New("Failed to Receive Available Devices for the Virtual Machine")
	}
	DiskController, ControllerError := Devices.FindDiskController("scsi")

	if ControllerError != nil {
		ErrorLogger.Printf(ControllerError.Error())
		return nil, errors.New("Failed to Receive DiskController for the Virtual Machine")
	}

	// Assigning Resource Configurations
	Devices.AssignController(DiskStorageConfig.Device, DiskController)
	defaults := types.VirtualMachineConfigSpec{
		GuestId:             vm.Guest.GuestId,
		NumCPUs:             ResourceConfig.NumCPUs,
		NumCoresPerSocket:   ResourceConfig.NumCoresPerSocket,
		MemoryMB:            ResourceConfig.MemoryMB,
		Version:             esx.HardwareVersion,
		CpuHotAddEnabled:    ResourceConfig.CpuHotAddEnabled,
		MemoryHotAddEnabled: ResourceConfig.MemoryHotAddEnabled,
		Firmware:            string(types.GuestOsDescriptorFirmwareTypeBios),
		DeviceChange:        []types.BaseVirtualDeviceConfigSpec{DiskStorageConfig},
	}

	// Setting up Configure Response Timeout on 5 mins...
	ConfigureTimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*5)
	defer CancelFunc()

	// Applying Configurations to the VM Server

	ConfigureTask, ConfiguredError := object.NewReference(&this.VimClient, vm.Reference()).(*object.VirtualMachine).Reconfigure(ConfigureTimeoutContext, defaults)
	HostSystemCustomizationTask, HostSystemCustomizationError := object.NewReference(&this.VimClient, vm.Reference()).(*object.VirtualMachine).Customize(ConfigureTimeoutContext, HostSystemCustomizationConfig)
	NetworkCustomizationTask, NetworkCustomizationError := object.NewReference(&this.VimClient, vm.Reference()).(*object.VirtualMachine).Customize(ConfigureTimeoutContext, *NetworkConfig)

	if ConfiguredError != nil {
		// If Failing To Apply First Configuration, Destroying Virtual Machine
		ErrorLogger.Printf("Failed to Configure Virtual Machine, Error has Occurred")
		_, Error := this.DestroyVirtualMachine(VirtualMachine)
		return nil, Error
	}

	if HostSystemCustomizationError != nil {
		ErrorLogger.Printf("Failed to Apply Customization Specification to the VM Server with OS Specifications, Error: %s", HostSystemCustomizationError)
		return nil, HostSystemCustomizationError
	}

	if NetworkCustomizationError != nil {
		ErrorLogger.Printf("Failed to Setup Customized Network")
		return nil, NetworkCustomizationError
	}

	// Waiting for Hardware and Resource Configuration to Apply
	WaitResponseError := ConfigureTask.Wait(ConfigureTimeoutContext)
	if WaitResponseError != nil {
		ErrorLogger.Printf("Failed to Configure Virtual Machine, Error: %s", WaitResponseError)
		return nil, WaitResponseError
	}
	// Waiting for OS Customization Response
	WaitCustomizationResponseError := HostSystemCustomizationTask.Wait(ConfigureTimeoutContext)
	if WaitCustomizationResponseError != nil {
		ErrorLogger.Printf("Failed to Configure OS Customization Specification for the VM Server, Error: %s", WaitCustomizationResponseError)
		return nil, WaitCustomizationResponseError
	}
	// Waiting for the Network Customization Response
	WaitNetworkCustomizationError := NetworkCustomizationTask.Wait(ConfigureTimeoutContext)
	if WaitNetworkCustomizationError != nil {
		ErrorLogger.Printf("Failed to Apply Network Configuration, Error: %s", WaitNetworkCustomizationError)
		return nil, WaitNetworkCustomizationError
	}
	// Applying SSH Credentials to the Virtual Machine Server....

	NewVirtualMachine := object.NewVirtualMachine(&this.VimClient, vm.Reference())

	VmIPAddress, VmIPError := NewVirtualMachine.WaitForIP(TimeoutContext, true)
	if VmIPError != nil {
		ErrorLogger.Printf("Failed to Fetch IP Addresses for the VM, Errors: [%s]", VmIPError)
	}

	// Setting up SSH Credentials for the Virtual Machine

	SSHManager := ssh_config.NewVirtualMachineSshManager(this.VimClient, VirtualMachine)
	PublicKey, PrivateKey, SSHError := SSHManager.GenerateSshKeys()
	if SSHError != nil {
		ErrorLogger.Printf("Failed to Generate SSH Keys for the VM, Error: %s", SSHError)
		return nil, SSHError
	}
	UploadSSHError := SSHManager.UploadSshKeys(*PrivateKey)
	if UploadSSHError != nil {
		ErrorLogger.Printf(
			"Failed to Upload SSH Private Key to the VM OS, Error: %s", UploadSSHError)
		return nil, UploadSSHError
	}

	// Installing Initial Dependencies on the Virtual Machine

	// DepInstaller := dependency_installer.NewEnviromentDependencyInstaller()
	// SshConnection := DepInstaller.GetSshConnection(IPAddress)
	// Installed, InstallError := DepInstaller.InstallDependencies(VirtualMachine)
	// if InstallError != nil {return nil, InstallError}

	return &VmInfo{
		IPAddress:    VmIPAddress,
		SshPublicKey: *PublicKey,
	}, nil
}

func (this *VirtualMachineManager) StartVirtualMachine(VirtualMachine *object.VirtualMachine) error {

	// Starts Virtual Machine Server..

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Newtask, DeployError := VirtualMachine.PowerOn(TimeoutContext)
	AppliedError := Newtask.Wait(TimeoutContext)

	switch {
	case DeployError != nil || AppliedError != nil:
		ErrorLogger.Printf("Failed to Start Virtual Machine, Errors: [%s, %s]",
			DeployError, AppliedError)
		return exceptions.VMDeployFailure()

	case DeployError == nil && AppliedError == nil:
		DebugLogger.Printf("Virtual Machine with ItemPath: %s, has been Started Successfully", VirtualMachine.InventoryPath)
		return nil
	default:
		return nil
	}
}

func (this *VirtualMachineManager) RebootVirtualMachine(VirtualMachine *object.VirtualMachine) bool {
	// Rebooting Virtual Machine Server and Operational System within this VM

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	RebootError := VirtualMachine.RebootGuest(TimeoutContext)
	if RebootError != nil {
		ErrorLogger.Printf("Failed to Reboot Guest OS, on VM: %s",
			VirtualMachine.Reference().Value)
		return true
	} else {
		return false
	}
}

func (this *VirtualMachineManager) ShutdownVirtualMachine(VirtualMachine *object.VirtualMachine) error {
	// Shutting Down Virtual Machine Server...

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Newtask, DeployError := VirtualMachine.PowerOff(TimeoutContext)
	AppliedError := Newtask.Wait(TimeoutContext)

	switch {
	case DeployError != nil || AppliedError != nil:
		ErrorLogger.Printf("Failed to Shutdown Virtual Machine, with ItemPath: %s Errors: [%s, %s]",
			VirtualMachine.InventoryPath, DeployError, AppliedError)
		return exceptions.VMShutdownFailure()

	case DeployError == nil && AppliedError == nil:
		DebugLogger.Printf("Virtual Machine with ItemPath: %s, has been Shutdown.", VirtualMachine.InventoryPath)
		return nil
	default:
		ErrorLogger.Printf("Unknown State has been Occurred, while Shutting Down Virtual Machine with ItemPath: %s",
			VirtualMachine.InventoryPath)
		return nil
	}
}

func (this *VirtualMachineManager) DestroyVirtualMachine(VirtualMachine *object.VirtualMachine) (bool, error) {
	// Destroys Virtual Machine, Customer Decided to get rid of...

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	DestroyTask, DestroyError := VirtualMachine.Destroy(TimeoutContext)
	DeletedError := DestroyTask.Wait(TimeoutContext)

	if DestroyError != nil || DeletedError != nil {
		ErrorLogger.Printf("Failed to Destroy Virtual Machine with ItemPath %s, Errors: [%s, %s]",
			VirtualMachine.InventoryPath, DestroyError, DeletedError)
		return false, exceptions.DestroyFailure()
	}
	InfoLogger.Printf("Virtual Machine with ItemPath: %s has been Destroyed",
		VirtualMachine.InventoryPath)
	return true, nil
}

// func (this *VirtualMachineManager) ReplicateVirtualMachine(TimeoutContext context.Context, VirtualMachine *object.VirtualMachine, VirtualMachineId string) (*VmInfo, error) {
// 	// Replicates Virtual Machine Server

// 	var MoVirtualMachine mo.VirtualMachine
// 	var NewVirtualMachine object.VirtualMachine
// 	var VirtualMachineConfiguration models.VirtualMachineConfiguration 
// 	var NativeConfiguration parsers.VirtualMachineCustomSpec

// 	Collector := property.DefaultCollector(&this.VimClient)
// 	RetrieveError := Collector.RetrieveOne(TimeoutContext, 
// 	VirtualMachine.Reference(), []string{"*"}, &MoVirtualMachine)


// 	var VirtualMachineModel models.VirtualMachine 
// 	var Datacenter *object.Datacenter
// 	var Datastore *object.Datastore 
// 	var Network *object.Network 
// 	var HostSystem *object.HostSystem 
// 	var ResourcePool *object.ResourcePool 
// 	var Folder *object.Folder 
// 	var ClusterComputeResource *object.ClusterComputeResource

// 	DatastoreRef := Datastore.Reference() 
// 	VmRef := NewVirtualMachine.Reference()
// 	ResourcePoolRef := ResourcePool.Reference() 
// 	HostSystemRef := HostSystem.Reference() 
// 	FolderRef := Folder.Reference()


// 	VmError := models.Database.Model(&models.VirtualMachine{}).Where(
// 	"id = ?", VirtualMachineId).Find(&VirtualMachineModel)
// 	if VmError.Error != nil {return nil, VmError.Error}

// 	if RetrieveError != nil {ErrorLogger.Printf(
// 	"Failed to Fetch Virtual Machine Mo Entity: Error: %s", RetrieveError); return nil, RetrieveError}
		
// 	Finder := find.NewFinder(&this.VimClient)
// 	Network = object.NewReference(&this.VimClient, MoVirtualMachine.Network[0]).(*object.Network) 
// 	ReplicaCounts := 5 


// 	// Obtaining Virtual Machine Configuration, which we are going to copy eventually 
// 	models.Database.Model(&models.VirtualMachineConfiguration{}).Where("virtual_machine_id = ?", VirtualMachineId).Find(&VirtualMachineConfiguration)
// 	// Converting to parsers.VirtualMahchineCustomSpec as this type is required at the following potential steps 
// 	VirtualMachineConfiguration.Scan(NativeConfiguration)

// 	if DatacenterRef, FindError := Finder.DefaultDatacenter(TimeoutContext); FindError != nil {
// 	return nil, FindError 
// 	}else{
// 		// Picking Up Datastore Depending on the Virtual Machine Hardware Configuration 
// 		DatacenterError := DatacenterRef.Properties(TimeoutContext,
// 	    DatacenterRef.Reference(), []string{"name"}, &Datacenter)

// 		if DatacenterError != nil {ErrorLogger.Printf(
// 		"Failed to Pick up available Datacenter, Error: %s", DatacenterError); return nil, DatacenterError}

// 		StorageReplicateConfiguration := types.StoragePlacementSpec{
// 			Vm: &VmRef, 
// 			ResourcePool: &ResourcePoolRef, 
// 			Host: &HostSystemRef, 
// 			Folder: &FolderRef, 
// 		}
// 		StoragePodResourceManager := object.NewStorageResourceManager(&this.VimClient) 
// 		Datastores, FindError := StoragePodResourceManager.RecommendDatastores(
// 		TimeoutContext, StorageReplicateConfiguration)

// 		if FindError != nil {}

// 		if len(Datastores.Recommendations) == 0 {DatastoreRef, DatastoreFindError := Finder.DefaultDatastore(
// 		TimeoutContext); Datastore = DatastoreRef; if DatastoreFindError != nil {return nil, DatastoreFindError}}

// 		Datastore = object.NewReference(&this.VimClient, Datastores.Recommendations[0].Action[0].(
// 		*types.StoragePlacementAction).Destination).(*object.Datastore)
// 	}
// 	Folders, FindError := Datacenter.Folders(TimeoutContext)

// 	if FindError != nil {ErrorLogger.Printf(
// 	"Failed to Find Available Folders for the Application, Error: %s", FindError)}


// 	VirtualMachineReplicateConfiguration := types.VirtualMachineRelocateSpec{
// 		Datastore: &DatastoreRef, 
// 		Folder: &FolderRef, 
// 		Pool: &ResourcePoolRef,
// 	}

// 	CloneConfiguration := types.VirtualMachineCloneSpec{
// 		Location: VirtualMachineReplicateConfiguration, 
// 		PowerOn: false, 
// 		Template: MoVirtualMachine.Config.Template,
// 	}
// 	// Initializing new Virtual Machine Instance ..... 

// 	InitializedVirtualMachine, InitializedError := this.InitializeNewVirtualMachine(
// 	this.VimClient, VirtualMachine.Name() + fmt.Sprintf("_%s", ReplicaCounts + 1),
//     Datastore, Network, ClusterComputeResource, object.NewReference(&this.VimClient, Folders.VmFolder.Reference()).(*object.Folder))

// 	// Applying Custom Configuration, catched from the Main VM Server Instance 
// 	switch InitializedError {
// 	case nil:
// 		ReplicatedVirtualMachineConfigInfo, ApplyException := this.ApplyConfiguration(InitializedVirtualMachine, NativeConfiguration)
// 		return ReplicatedVirtualMachineConfigInfo, ApplyException 
// 	default:
// 		ErrorLogger.Printf("Failed to Initialize Base Virtual Machine, Error: %s", InitializedError)
// 		return nil, InitializedError
// 	}
// }
