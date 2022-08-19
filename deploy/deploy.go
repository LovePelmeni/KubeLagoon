package deploy

import (
	"context"
	"errors"
	"log"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/exceptions"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/parsers"

	"github.com/LovePelmeni/Infrastructure/resources"
	"github.com/LovePelmeni/Infrastructure/storage"

	"github.com/LovePelmeni/Infrastructure/suggestions"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25"
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

	var VirtualMachineConfiguration models.Configuration

	VirtualMachineGormRef := models.Database.Model(
		&models.VirtualMachine{}).Where(
		"owner_id = ? AND id = ?",
		CustomerId, VmId).Association(
		"Configuration").Find(&VirtualMachineConfiguration)

	if VirtualMachineGormRef.Error != nil {
		ErrorLogger.Printf("Failed to Find Virtual Machine in Database with ID: %s and Owner ID: %s", VmId, CustomerId)
		return nil, exceptions.ItemDoesNotExist()
	}

	// Receiving Prepared Virtual Machine Instance by using API

	FinderIndex := object.NewSearchIndex(&this.VimClient)
	VirtualRef, FindError := FinderIndex.FindByInventoryPath(TimeoutContext, VirtualMachineConfiguration.ItemPath)

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

func (this *VirtualMachineManager) DeployVirtualMachine(
	VimClient vim25.Client,
	HardwareConfiguration parsers.HardwareConfig,
	CustomConfiguration parsers.VirtualMachineCustomSpec,

) (*object.VirtualMachine, error) {

	// Initializing New Virtual Machine

	Resources, AvailableError := HardwareConfiguration.GetResources(VimClient)

	if AvailableError != nil {
		return nil,
			errors.New("Sorry, But Not All Resources is Currently Available to Deploy Your Server :(")
	}

	InitializedMachine, InitError := this.InitializeNewVirtualMachine(
		VimClient, CustomConfiguration.Metadata.VirtualMachineName, Resources["Datastore"].(*object.Datastore),
		Resources["Datacenter"].(*object.Datacenter), Resources["Network"].(*object.Network),
		Resources["ResourcePool"].(*object.ResourcePool), Resources["Folder"].(*object.Folder),
	)

	if InitError != nil {
		ErrorLogger.Printf("Failed to Initialize New Virtual Machine Instance")
		return nil, errors.New("Failed to Initialize New Virtual Server")
	}

	// Applying Configuration

	ApplyError := this.ApplyConfiguration(InitializedMachine, CustomConfiguration, Resources["Datastore"].(*object.Datastore))
	if ApplyError != nil {
		ErrorLogger.Printf("Failed to Apply Custom Configuration to the VM: %s",
			InitializedMachine.Reference().Value)
		return nil, errors.New("Failed to Apply Virtual Server Custom Configuration")
	}

	// Starting Virtual Machine Server
	StartedError := this.StartVirtualMachine(InitializedMachine)
	if StartedError != nil {
		ErrorLogger.Printf("Failed to Start New VM: %s, Error: %s",
			InitializedMachine.Reference().Value, StartedError)
		return nil, errors.New("Failed to Start New Virtual Server")
	}
	return InitializedMachine, nil
}

func (this *VirtualMachineManager) InitializeNewVirtualMachine(
	VimClient vim25.Client,
	VirtualMachineName string,
	DataStore *object.Datastore, // Name, Customer Decided to set up for this Virtual Server
	Datacenter *object.Datacenter,
	Network *object.Network,
	ResourcePool *object.ResourcePool,
	Folder *object.Folder,
) (*object.VirtualMachine, error) {
	// Initializes Virtual Machine Configuration (That does not exist yet)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	ResourceAllocationManager := NewVirtualMachineResourceKeyManager()
	ResourceCredentials, ResourceKeysError := ResourceAllocationManager.GetResourceKeys(
		ResourcePool, Folder)

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
					Value: Network.Reference().Value,
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
				VmConfigSpec:        &vcenter.VmConfigSpec{},
				StorageProvisioning: "thin",
			},
			Target: vcenter.Target{
				ResourcePoolID: ResourcePool.Reference().Value,
				FolderID:       Folder.Reference().Value,
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

func (this *VirtualMachineManager) ApplyConfiguration(
	VirtualMachine *object.VirtualMachine,
	Configuration parsers.VirtualMachineCustomSpec,
	Datastore *object.Datastore,
) error {
	// Applies Custom Configuration: Num's of CPU's, Memory etc... onto the Initialized Virtual Machine

	// Resource Manager, that allows to return Initialized Resource Instances, by ID, UniqueName, etc...
	ResourceManager := suggestions.NewResourceSuggestManager(this.VimClient)

	// Receiving Datastore Instance by InventoryPath
	DataStore, _ := ResourceManager.GetResource(Datastore.InventoryPath)
	DataStoreManagedReference := DataStore.Reference()

	// Initializing Configurations for the Virtual Server
	DiskConfig, DiskError := storage.NewVirtualMachineStorageManager().SetupStorageDisk(VirtualMachine, *storage.NewVirtualMachineStorage(Configuration.Disk.CapacityInKB), &DataStoreManagedReference)
	ResourceConfig, ResourceError := resources.NewVirtualMachineResourceManager().SetupResources(resources.NewVirtualMachineResources(Configuration.Resources.CpuNum, int64(Configuration.Resources.MemoryInMegabytes)))

	// If Failed to Obtain one of the Hardware Resources, Returns Exception, that it Does Not Exist

	if DiskError != nil {
		return exceptions.ComponentDoesNotExist(DiskError.Error())
	}
	if ResourceError != nil {
		return exceptions.ComponentDoesNotExist(ResourceError.Error())
	}

	// Applying Hardware Configurations, such as CPU, Memory, Disk etc....

	for _, VmConfig := range []types.VirtualMachineConfigSpec{*DiskConfig, *ResourceConfig} {

		HardwareError := func() error {
			TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
			defer CancelFunc()

			NewApplyTask, ApplyError := VirtualMachine.Reconfigure(TimeoutContext, VmConfig)
			WaitError := NewApplyTask.Wait(TimeoutContext)

			switch {
			case ApplyError != nil || WaitError != nil:
				ErrorLogger.Printf(
					"Hardware Configuration has been Failed to be Applied! For VM: %s",
					VirtualMachine.Reference().Value)
				return ApplyError

			case ApplyError == nil && WaitError == nil:
				DebugLogger.Printf(
					"Hardware Configuration has been Applied Successfully! For VM: %s",
					VirtualMachine.Reference().Value)
				return nil

			default:
				return ApplyError
			}
		}()
		if HardwareError != nil {
			return HardwareError
		} else {
			continue
		}
	}

	// Applying Customization Configurations, such as IP, Profiles, etc...

	for _, VmCustomizationConfig := range []types.CustomizationSpec{} {
		CustomizationError := func() error {

			TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
			defer CancelFunc()

			NewApplyTask, ApplyError := VirtualMachine.Customize(TimeoutContext, VmCustomizationConfig)
			WaitError := NewApplyTask.Wait(TimeoutContext)

			switch {
			case ApplyError != nil || WaitError != nil:
				ErrorLogger.Printf(
					"Customized Configuration has been Failed to be Applied! For VM: %s",
					VirtualMachine.Reference().Value)
				return ApplyError

			case ApplyError == nil && WaitError == nil:
				DebugLogger.Printf(
					"Customized Configuration has been Applied Successfully! For VM: %s",
					VirtualMachine.Reference().Value)
				return nil
			default:
				return ApplyError
			}
		}()
		if CustomizationError != nil {
			return CustomizationError
		} else {
			continue
		}
	}
	return nil
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
