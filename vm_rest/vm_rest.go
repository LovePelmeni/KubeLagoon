package vm_rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/parsers"
	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/rest"
)

// Insfrastructure API Environment Variables
var (
	APIIp    = os.Getenv("VMWARE_SOURCE_IP")
	Username = os.Getenv("VMWARE_SOURCE_USERNAME")
	Password = os.Getenv("VMWARE_SOURCE_PASSWORD")

	APIUrl = &url.URL{
		Scheme: "https",
		Path:   "/sdk/",
		Host:   APIIp,
		User:   url.UserPassword(Username, Password),
	}
)

var (
	Client govmomi.Client
)

var (
	Customer models.Customer
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {

	LogFile, Error := os.OpenFile("RestVm.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if Error != nil {
		panic(Error)
	}

	DebugLogger = log.New(LogFile, "DEBUG:", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO:", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR:", log.Ldate|log.Ltime|log.Lshortfile)

	var RestClient *rest.Client
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	APIClient, ConnectionError := govmomi.NewClient(TimeoutContext, APIUrl, false)
	switch {
	case ConnectionError != nil:
		panic(ConnectionError)

	case ConnectionError == nil:
		RestClient = rest.NewClient(APIClient.Client)
		if FailedToLogin := RestClient.Login(TimeoutContext, APIUrl.User); FailedToLogin != nil {
			panic(FailedToLogin)
		}
	}
	Client = *APIClient
}

// Package which Contains Rest API Controllers, for Handling VM's Behaviour

// VM Rest API Controllers

func GetCustomerVirtualMachine(RequestContext *gin.Context) {
	// Returns Extended Info about Virtual Machine Server Owned by the Customer

	var VirtualMachine models.VirtualMachine
	CustomerId := RequestContext.Query("CustomerId")
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	Gorm := models.Database.Model(&VirtualMachine).Where(
		"owner_id = ? AND id = ?", CustomerId, VirtualMachineId).Find(&VirtualMachine)

	switch Gorm.Error {
	case nil:
		RequestContext.JSON(http.StatusOK,
			gin.H{"VirtualMachine": VirtualMachine})
	default:
		ErrorLogger.Printf("Failed to Receive Virtual Machine, Error: %s", Gorm.Error.Error())
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Machine Does Not Exist"})
	}
}

func GetCustomerVirtualMachines(RequestContext *gin.Context) {
	// Returns List of the VM's that Customer Owns
	var VirtualMachines []models.VirtualMachine
	CustomerId := RequestContext.Query("CustomerId")
	Gorm := models.Database.Model(&Customer).Where("id = ?", CustomerId).Preload("Vms").Find(&VirtualMachines)
	switch Gorm.Error {
	case nil:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": VirtualMachines})
	default:
		ErrorLogger.Printf("Failed to Receive All Customer Virtual Machines, Error: %s", Gorm.Error)
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": fmt.Sprintf("%s", Gorm.Error)})
	}
}

// Virtual Machine Rest API Endpoints

func DeployNewVirtualMachineRestController(RequestContext *gin.Context) {

	// Rest Controller, that deploys new Virtual Machine with Custom Configuration

	CustomerId := RequestContext.PostForm("customerId")

	HardwareConfiguration, HardwareError := parsers.NewHardwareConfig(RequestContext.PostForm("HardwareConfiguration"))
	CustomConfiguration, CustomError := parsers.NewCustomConfig(RequestContext.PostForm("CustomConfiguration"))

	if HardwareError != nil || CustomError != nil {
		RequestContext.JSON(
			http.StatusBadRequest, gin.H{"Error": "Invalid Configuration"})
	}

	// Deploying New Virtual Server Instance

	NewVirtualMachineManager := deploy.NewVirtualMachineManager(*Client.Client)
	DeployedInstance, DeployError := NewVirtualMachineManager.DeployVirtualMachine(
		*Client.Client, *HardwareConfiguration, *CustomConfiguration)

	// Creating New ORM Object
	VirtualMachine, _ := models.NewVirtualMachine(CustomerId,
		CustomConfiguration.Metadata.VirtualMachineName, DeployedInstance.InventoryPath).Create()

	switch DeployError {

	case nil:
		VirtualMachine.Commit()

		RequestContext.JSON(http.StatusOK,
			gin.H{"Operation": "Success"})
	default:
		VirtualMachine.Rollback()
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Deploy New Virtual Server"})
	}
}

func UpdateVirtualMachineConfigurationRestController(RequestContext *gin.Context) {
	// Rest Controller, that is Used to Apply New Configuration to the Virtual Machine Server
}

func StartVirtualMachineRestController(RequestContext *gin.Context) {
	// Rest Controller, that is Used to Start Virtual Machine Server
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := RequestContext.Query("CustomerId")

	VmManager := deploy.NewVirtualMachineManager(*Client.Client)
	Vm, VmError := VmManager.GetVirtualMachine(VirtualMachineId, CustomerId)

	if VmError != nil {
		RequestContext.JSON(http.StatusBadRequest, gin.H{"Error": "VM Does not Exist."})
	}
	StartedError := VmManager.StartVirtualMachine(Vm)

	switch StartedError {
	case nil:
		RequestContext.JSON(http.StatusOK, gin.H{"Status": "Started"})
	default:
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": StartedError})
	}
}

func RebootVirtualMachineRestController(RequestContext *gin.Context) {
	// Rest Controller, that is Used to Reboot Virtual Machine Server
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := RequestContext.Query("CustomerId")
	NewVmManager := deploy.NewVirtualMachineManager(*Client.Client)
	Vm, VmError := NewVmManager.GetVirtualMachine(VirtualMachineId, CustomerId)

	if VmError != nil {
		RequestContext.JSON(
			http.StatusBadRequest, gin.H{"Error": "VM Does Not Exist."})
		return
	}

	Rebooted := NewVmManager.RebootVirtualMachine(Vm)

	switch Rebooted {
	case true:
		RequestContext.JSON(http.StatusOK, gin.H{"Status": "Rebooted"})
	case false:
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": VmError})
	}
}

func ShutdownVirtualMachineRestController(RequestContext *gin.Context) {

	// Rest Controller, that Is Used to Shutdown Virtual Machine Server

	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := RequestContext.Query("CustomerId")
	NewVmManager := deploy.NewVirtualMachineManager(*Client.Client)

	Vm, FindError := NewVmManager.GetVirtualMachine(VirtualMachineId, CustomerId)

	if FindError != nil {
		RequestContext.AbortWithStatusJSON(
			http.StatusBadRequest, gin.H{"Error": "Server does Not Exist"})
		return
	}

	StartedError := NewVmManager.StartVirtualMachine(Vm)
	switch {
	case StartedError != nil:
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": fmt.Sprintf("Failed to Start the Server, %s", StartedError)})

	case StartedError == nil:
		RequestContext.JSON(http.StatusCreated, gin.H{"Operation": "Success"})
	}

}

func RemoveVirtualMachineRestController(RequestContext *gin.Context) {
	// Rest Controller, that is Used for Destroying Virtual machines...

	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := RequestContext.Query("CustomerId")
	NewVmManager := deploy.NewVirtualMachineManager(*Client.Client)

	Vm, FindError := NewVmManager.GetVirtualMachine(VirtualMachineId, CustomerId)

	if FindError != nil {
		RequestContext.AbortWithStatusJSON(
			http.StatusBadRequest, gin.H{"Error": "Server does Not Exist"})
		return
	}

	Started, StartedError := NewVmManager.DestroyVirtualMachine(Vm)

	switch {
	case StartedError != nil || Started != true:
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": fmt.Sprintf("Failed to Start the Server, %s", StartedError)})

	case StartedError == nil && Started:
		RequestContext.JSON(http.StatusCreated, gin.H{"Operation": "Success"})
	}
}
