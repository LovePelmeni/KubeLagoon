package deploy

import (
	"context"
	"log"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/vmware/govmomi/find"

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

type DeployResourceKeys struct {
	// Credentials
	// * For the Storage: `Storage Key` - represents Acccess Key to the Storage, where All the Data will Be Allocated.
	// * For the Network: `Network Key` - represents Access Key to the Network, the Application will be Attached to.
	NetworkKey string `json:"NetworkKey"`
	StorageKey string `json:"StorageKey"`
}

type VirtualMachineDeployKeyManagerInterface interface {
	// Interface, is used for obtaining necessary Resource Keys, in order To Initialize
	// Initial Virtual Machine Instance
	GetDeployResourceKeys(RestClient rest.Client) (DeployResourceKeys, error)
}

type VirtualMachineDeployerInterface interface {
	// Interface, that Deploys new Virtual Machine
	DeployVirtualMachine() *object.VirtualMachine
}

type VirtualMachineDeployKeyManager struct {

	// Class, that is responsible for Obtaining Credentials, to the Storage/Network
	// To Get the Permissions to use that Resources.

	VirtualMachineDeployKeyManagerInterface
}

func NewVirtualMachineDeployKeyManager() *VirtualMachineDeployKeyManager {
	return &VirtualMachineDeployKeyManager{}
}
func (this *VirtualMachineDeployKeyManager) GetDeployResourceKeys() (DeployResourceKeys, error) {
}

type VirtualMachineDeployer struct {
	VirtualMachineDeployerInterface
}

func NewVirtualMachineDeployer() *VirtualMachineDeployer {
	return &VirtualMachineDeployer{}
}

func (this *VirtualMachineDeployer) InitializeNewVirtualMachine(
	APIClient rest.Client, // Rest Client for API Calls
	Datastore *object.Datastore, // Datastore, that has been chosen by Customer where the want to Store Data
	Datacenter *object.Datacenter, // Datacenter, that has been chosen by Customer, where they want to deploy their Application
	Network *object.Network, // Network, that has been chosen By Customer, where they want to attach their Application At,
	Folder *object.Folder, // Folder, where the Application Item is going to be Stored.
	Resource *object.ResourcePool, // Resource, (Memory and CPU Num's) that Customer want to Allocate, according to their Requirements

) (*object.VirtualMachine, error) {
	// Initializes Virtual Machine Configuration

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
						Type:        "DATASTORE",
						DatastoreID: Datastore.Reference().Value,

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
			newFinder := find.NewFinder(APIClient)
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
		DebugLogger.Printf("No Resources Is Available.")
		return nil, exceptions.NoResourceAvailable()

	default:
		DebugLogger.Printf("Unknown Deploy Errors: [%s]", ResourceKeysError)
		return nil, exceptions.VMDeployFailure()
	}
	return nil, exceptions.VMDeployFailure()
}

func (this *VirtualMachineDeployer) StartVirtualMachine(VirtualMachine *object.VirtualMachine) error {

	// Starts Virtual Machine Server...
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Newtask, DeployError := VirtualMachine.PowerOn(TimeoutContext)
	AppliedError := Newtask.Wait(TimeoutContext)

	switch {
	case DeployError != nil || AppliedError != nil:
		ErrorLogger.Printf("Failed to Start Virtual Machine, Errors: [%s, %s]", DeployError, AppliedError)
		return exceptions.VMDeployFailure()

	case DeployError == nil && AppliedError == nil:
		DebugLogger.Printf("Application has been Deployed Successfully.")
		return nil
	default:
		return nil
	}

}

func (this *VirtualMachineDeployer) ShutdownVirtualMachine(VirtualMachine *object.VirtualMachine) error {

	// Shutting Down Virtual Machine Server...
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Newtask, DeployError := VirtualMachine.PowerOff(TimeoutContext)
	AppliedError := Newtask.Wait(TimeoutContext)

	switch {
	case DeployError != nil || AppliedError != nil:
		ErrorLogger.Printf("Failed to Start Virtual Machine, Errors: [%s, %s]", DeployError, AppliedError)
		return exceptions.VMShutdownFailure()

	case DeployError == nil && AppliedError == nil:
		DebugLogger.Printf("Application has been Deployed Successfully.")
		return nil
	default:
		return nil
	}
}
