package deploy

import (
	"context"
	"log"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/parsers"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25"

	"github.com/vmware/govmomi/object"
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

type VirtualMachineDeployKeyManagerInterface interface {
	// Interface, is used for obtaining necessary Resource Keys, in order To Initialize
	// Initial Virtual Machine Instance
	GetDeployResourceKeys(RestClient rest.Client) (ResourceKeys, error)
}

type VirtualMachineManagerInterface interface {
	// Interface, that Deploys new Virtual Machine

	StartVirtualMachine(VmId string, CustomerId string) (bool, error)

	ShutdownVirtualMachine(VmId string, CustomerId string) (bool, error)

	ApplyConfiguration(VmId string, CustomerId string, Configuration parsers.Config) (bool, error)

	InitializeNewVirtualMachine(
		VimClient vim25.Client, APIClient rest.Client,
		Datastore *object.Datastore, Datacenter *object.Datacenter,
		Folder *object.Folder, ResourcePool *object.ResourcePool,
	) *object.VirtualMachine

	DestroyVirtualMachine(VmId string, CustomerId string) (bool, error)
}

type ResourceKeys struct {
	// Credentials
	// * For the Storage: `Storage Key` - represents Acccess Key to the Storage, where All the Data will Be Allocated.
	// * For the Network: `Network Key` - represents Access Key to the Network, the Application will be Attached to.
	NetworkKey string `json:"NetworkKey"`
	StorageKey string `json:"StorageKey"`
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

	VirtualMachineDeployKeyManagerInterface
}

func NewVirtualMachineDeployKeyManager() *VirtualMachineResourceKeyManager {
	return &VirtualMachineResourceKeyManager{}
}

func (this *VirtualMachineResourceKeyManager) GetResourceKeys() (ResourceKeys, error) {
	// Returns Storage and Resource Key in order to get Usage Permissions

}

type VirtualMachineManager struct {

	// Class, that Is Taking care of the Virtual Machine Deployment Process
	// It is responsible for Deploying/ Starting / Stopping / Updating Virtual Machines
	// Owned by Customers
	// Provides Following Methods in order to Fullfill the Needs and make the Process comfortable and easier
	VirtualMachineManagerInterface
	VimClient vim25.Client
}

func NewVirtualMachineDeployer() *VirtualMachineManager {
	return &VirtualMachineManager{}
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

func (this *VirtualMachineManager) InitializeNewVirtualMachine(
	VimClient vim25.Client,
	APIClient rest.Client, // Rest Client for API Calls
	Datastore *object.Datastore, // Datastore, that has been chosen by Customer where the want to Store Data
	Datacenter *object.Datacenter, // Datacenter, that has been chosen by Customer, where they want to deploy their Application
	Network *object.Network, // Network, that has been chosen By Customer, where they want to attach their Application At,
	Folder *object.Folder, // Folder, where the Application Item is going to be Stored.
	Resource *object.ResourcePool, // Resource, (Memory and CPU Num's) that Customer want to Allocate, according to their Requirements

) (*object.VirtualMachine, error) {
	// Initializes Virtual Machine Configuration (That does not exist yet)

	ResourceAllocationManager := NewVirtualMachineDeployKeyManager()
	ResourceCredentials, ResourceKeysError := ResourceAllocationManager.GetDeployResourceKeys()

	switch {

	// If Resources Keys has been Received, Going through the Next Steps
	case ResourceKeysError == nil && len(ResourceCredentials.NetworkKey) != 0 && len(ResourceCredentials.StorageKey) != 0:

		Deployment := vcenter.Deploy{
			DeploymentSpec: vcenter.DeploymentSpec{

				Name:               "test",
				DefaultDatastoreID: Datastore.Reference().Value,
				AcceptAllEULA:      true,
				NetworkMappings: []vcenter.NetworkMapping{{
					Key:   ResourceCredentials.NetworkKey,
					Value: Network.Reference().Value,
				}},

				StorageMappings: []vcenter.StorageMapping{{
					Key: ResourceCredentials.StorageKey,
					Value: vcenter.StorageGroupMapping{
						Type:         "DATASTORE",
						DatastoreID:  Datastore.Reference().Value,
						Provisioning: "thin",
					},
				}},
				StorageProvisioning: "thin",
			},
			Target: vcenter.Target{
				ResourcePoolID: Resource.Reference().Value,
				FolderID:       Folder.Reference().Value,
			},
		}
		DeployTimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
		defer CancelFunc()

		VirtualMachineInstanceReference, DeployError := vcenter.NewManager(&APIClient).DeployLibraryItem(DeployTimeoutContext, item.ID, Deployment)

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

func (this *VirtualMachineManager) ApplyConfiguration(VmId string, CustomerId string, Configuration parsers.Config) {
	// Applies Custom Configuration: Num's of CPU's, Memory etc... onto the Initialized Virtual Machine

	// Initializing Applier Managers
	// Configurations
}

func (this *VirtualMachineManager) StartVirtualMachine(VmId string, CustomerId string) error {

	// Starts Virtual Machine Server..

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	VirtualMachine, VmError := this.GetVirtualMachine(VmId, CustomerId)
	if VmError != nil {
		return VmError
	}

	Newtask, DeployError := VirtualMachine.PowerOn(TimeoutContext)
	AppliedError := Newtask.Wait(TimeoutContext)

	switch {
	case DeployError != nil || AppliedError != nil:
		ErrorLogger.Printf("Failed to Start Virtual Machine, with ID: %s, of Owner: %s Errors: [%s, %s]",
			VmId, CustomerId, DeployError, AppliedError)
		return exceptions.VMDeployFailure()

	case DeployError == nil && AppliedError == nil:
		DebugLogger.Printf("Virtual Machine with ID: %s of Owner: %s has been Started Successfully", VmId, CustomerId)
		return nil
	default:
		return nil
	}

}

func (this *VirtualMachineManager) ShutdownVirtualMachine(VmId string, CustomerId string) error {
	// Shutting Down Virtual Machine Server...

	VirtualMachine, ExistsError := this.GetVirtualMachine(VmId, CustomerId)

	if ExistsError != nil {
		return ExistsError
	}

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Newtask, DeployError := VirtualMachine.PowerOff(TimeoutContext)
	AppliedError := Newtask.Wait(TimeoutContext)

	switch {
	case DeployError != nil || AppliedError != nil:
		ErrorLogger.Printf("Failed to Shutdown Virtual Machine, with ID: %s of Owner: %s, Errors: [%s, %s]",
			VmId, CustomerId, DeployError, AppliedError)
		return exceptions.VMShutdownFailure()

	case DeployError == nil && AppliedError == nil:
		DebugLogger.Printf("Virtual Machine with ID: %s, Of Owner: %s has been Shutdown.", VmId, CustomerId)
		return nil
	default:
		ErrorLogger.Printf("Unknown State has been Occurred, while Shutting Down Virtual Machine with ID: %s of Owner: %s",
			VmId, CustomerId)
		return nil
	}
}

func (this *VirtualMachineManager) DestroyVirtualMachine(VmId string, CustomerId string) (bool, error) {
	// Destroys Virtual Machine, Customer Decided to get rid of...

	VirtualMachine, ExistsError := this.GetVirtualMachine(VmId, CustomerId)

	if ExistsError != nil {
		return false, ExistsError
	}

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	DestroyTask, DestroyError := VirtualMachine.Destroy(TimeoutContext)
	DeletedError := DestroyTask.Wait(TimeoutContext)

	if DestroyError != nil || DeletedError != nil {
		ErrorLogger.Printf("Failed to Destroy Virtual Machine with ID: %s of Owner: %s, Errors: [%s, %s]",
			VmId, CustomerId, DestroyError, DeletedError)
		return false, exceptions.DestroyFailure()
	}
	InfoLogger.Printf("Virtual Machine with ID: %s, Of Owner: %s has been Destroyed", VmId, CustomerId)
	return true, nil
}
