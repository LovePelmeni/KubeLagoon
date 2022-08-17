package deploy

import (
	"context"
	"log"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/vmware/govmomi/object"
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

type VirtualMachineDeployerInterface interface {
	// Interface, that Deploys new Virtual Machine
	DeployVirtualMachine() *object.VirtualMachine
}

type VirtualMachineDeployer struct {
	VirtualMachineDeployerInterface
}

func NewVirtualMachineDeployer() *VirtualMachineDeployer {
	return &VirtualMachineDeployer{}
}

func (this *VirtualMachineDeployer) InitializeVirtualMachine() *object.VirtualMachine {
	// Initializes Virtual Machine Configuration
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
