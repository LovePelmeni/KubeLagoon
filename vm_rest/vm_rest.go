package vm_rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/parsers"
	"github.com/LovePelmeni/Infrastructure/suggestions"
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

func InitializeVirtualMachineRestController(RequestContext *gin.Context) {

	//Rest Controller, that Initializes New Empty Virtual Machine

	VirtualMachineName := RequestContext.PostForm("VirtualMachineName")
	CustomerId := RequestContext.PostForm("CustomerId")

	// Initilizing Resource Requirements Instance, that will be used to pick up Appropriate Hardware Instances, based on this Requirements
	ResourceRequirements := suggestions.NewResourceRequirements(RequestContext.PostForm("ResourceRequirements"))

	// Initializing Hardware Configuration Based on the Resource Requirements
	HardwareConfig, ParseError := parsers.NewHardwareConfig(RequestContext.PostForm("DatacenterConfig"))
	if ParseError != nil {
		RequestContext.JSON(http.StatusBadRequest, gin.H{"Error": "Failed to Initialize New Virtual Server, Invalid Configuration has been Passed"})
		return
	}

	// Parsing the Resource Instances, based on the Hardwares Configuration and Resource Requirements
	// that has been Provided by the Customer Initially

	ParsedResourceInstances := HardwareConfig.ParseResources(*ResourceRequirements)
	if reflect.ValueOf(ParsedResourceInstances).IsNil() {
		RequestContext.JSON(
			http.StatusBadGateway, gin.H{"Error": "Failed to Get Resource Instances for Initializing New Server, Might be Some of the Instances does not Exist"})
		return
	}

	// Initializing New Virtual Server Instance...

	InstanceDeployer := deploy.NewVirtualMachineManager(*Client.Client)
	InitializedInstance, InitError := InstanceDeployer.InitializeNewVirtualMachine(
		*Client.Client, VirtualMachineName,
		ParsedResourceInstances["Datastore"],
		ParsedResourceInstances["Datacenter"],
		ParsedResourceInstances["Network"],
		ParsedResourceInstances["ResourcePool"],
		ParsedResourceInstances["Folder"],
	)

	switch InitError {
	case nil:
		NewVirtualMachine := models.NewVirtualMachine(
			CustomerId, VirtualMachineName, InitializedInstance.InventoryPath)
		NewVirtualMachine.Create()
		RequestContext.JSON(http.StatusCreated, gin.H{"Status": "Initialized"})
	default:
		ErrorLogger.Printf("Failed to Initialize New Virtual Server, Error: %s", InitError)
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Initialize new Virtual Server"})
	}
}

func DeployVirtualMachineRestController(RequestContext *gin.Context) {
	// Rest Controller, that Applies Configuration on the Existing Initialized Virtual Machine Server
	// Before Calling this Method, you firsly need to call `InitializeVirtualMachineRestController`.
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
