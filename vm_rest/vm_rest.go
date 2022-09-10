package vm_rest

import (
	"context"
	"encoding/json"
	"fmt"

	"net/http"

	"net/url"
	"os"

	"reflect"
	"strconv"
	"time"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/models"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/LovePelmeni/Infrastructure/parsers"
	"github.com/LovePelmeni/Infrastructure/resources"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/rest"
	_ "github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25/mo"
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
	Client *govmomi.Client
)

var (
	Customer models.Customer
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("VmRestLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {

	var RestClient *rest.Client
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	APIClient, ConnectionError := govmomi.NewClient(TimeoutContext, APIUrl, false)
	switch {
	case ConnectionError != nil:
		Logger.Error("FAILED TO INITIALIZE CLIENT, DOES THE VMWARE HYPERVISOR ACTUALLY RUNNING?")

	case ConnectionError == nil:
		RestClient = rest.NewClient(APIClient.Client)
		if FailedToLogin := RestClient.Login(TimeoutContext, APIUrl.User); FailedToLogin != nil {
			Logger.Error("FAILED TO LOGIN TO THE VMWARE HYPERVISOR SERVER", zap.Error(FailedToLogin))
		}
	}
	Client = APIClient
}

// Package which Contains Rest API Controllers, for Handling VM's Behaviour

// VM Rest API Controllers

func GetCustomerVirtualMachine(RequestContext *gin.Context) {
	// Returns Extended Info about Virtual Machine Server Owned by the Customer

	var VirtualMachine models.VirtualMachine
	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(
		RequestContext.Request.Header.Get("jwt-token"))

	CustomerId := jwtCredentials.UserId
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	Gorm := models.Database.Model(&VirtualMachine).Where(
		"owner_id = ? AND id = ?", CustomerId, VirtualMachineId).Find(&VirtualMachine)

	switch Gorm.Error {
	case nil:
		RequestContext.JSON(http.StatusOK,
			gin.H{"VirtualMachine": VirtualMachine})
	default:
		Logger.Error("Failed to Receive Virtual Machine, Error: %s", zap.Error(Gorm.Error))
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Machine Does Not Exist"})
	}
}

func GetCustomerVirtualMachines(RequestContext *gin.Context) {
	// Returns List of the VM's that Customer Owns

	var VirtualMachines []models.VirtualMachine
	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(
		RequestContext.Request.Header.Get("jwt-token"))

	CustomerId := jwtCredentials.UserId
	Gorm := models.Database.Model(&Customer).Where("id = ?", CustomerId).Preload("Vms").Find(&VirtualMachines)
	switch Gorm.Error {
	case nil:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": VirtualMachines})
	default:
		Logger.Error("Failed to Receive All Customer Virtual Machines", zap.Error(Gorm.Error))
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": fmt.Sprintf("%s", Gorm.Error)})
	}
}

// Virtual Machine Rest API Endpoints

func InitializeVirtualMachineRestController(RequestContext *gin.Context) {

	//Rest Controller, that Initializes New Empty Virtual Machine + Load Balancer

	// Receiving Extra Info, that is going to be Necessary to Initialize New VM Server

	JwtCookie, _ := authentication.GetCustomerJwtCredentials(
		RequestContext.Request.Header.Get("Authorization"))

	VirtualMachineName := RequestContext.PostForm("VirtualMachineName")
	CustomerId := JwtCookie.UserId

	// Initilizing Resource Requirements Instance, that will be used to pick up Appropriate Hardware Instances of the Choosed Datacenter, based on this Requirements
	DatacenterResourceRequirements, InvalidError := resources.NewDatacenterResourceRequirements(
		RequestContext.PostForm("ResourceRequirements"))

	// On Parse Failure Returning Bad Request with Error Explanation
	if InvalidError != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Invalid Configuration has been Passed."})
		return
	}

	// Initializing Hardware Configuration Based on the Resource Requirements
	DatacenterConfig, ParseError := parsers.NewHardwareConfig(RequestContext.PostForm("DatacenterConfig"))
	if ParseError != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Failed to Initialize New Virtual Server, Invalid Configuration has been Passed"})
		return
	}

	// Receiving Datacenter Instance, based on Obtained Datacenter Config

	Datacenter, FindError := DatacenterConfig.GetDatacenter(*Client.Client)
	if FindError != nil {
		RequestContext.JSON(http.StatusBadRequest, gin.H{"Error": FindError})
		return
	}

	// Initializing Datacenter Manager, to pick up Compute Resources, based on the Requirements
	DatacenterResourceManager := resources.NewDatacenterResourceManager(Client.Client)

	// Returns Components of the Datacenter, (Network, Datastore, Storage, Folder, etc...), that Matches Requirements, specified in the DatacenterResourceRequirements
	ParsedResourceInstances, FindError := DatacenterResourceManager.GetComputeResources(Datacenter, *DatacenterResourceRequirements)

	// Checking if Parsed Resource Instances is not Nil or Empty Slice....
	if reflect.ValueOf(ParsedResourceInstances).IsNil() || FindError != nil {
		RequestContext.JSON(
			http.StatusBadGateway, gin.H{"Error": FindError.Error()})
		return
	}

	// Initializing New Virtual Server Instance...

	InstanceDeployer := deploy.NewVirtualMachineManager(*Client.Client)
	InitializedInstance, InitError := InstanceDeployer.InitializeNewVirtualMachine(
		*Client.Client, VirtualMachineName,
		ParsedResourceInstances["Datastore"].(*object.Datastore),
		ParsedResourceInstances["Network"].(*object.Network),
		ParsedResourceInstances["ClusterComputeResource"].(*object.ClusterComputeResource),
		ParsedResourceInstances["Folder"].(*object.Folder),
	)

	switch InitError {
	case nil:
		// Creating New Virtual Machine Model ORM Object.... and store it into SQL DB

		TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
		defer CancelFunc()

		IPAddress, IPError := InitializedInstance.WaitForIP(TimeoutContext)
		if IPError != nil {
			Logger.Error(
				"Failed to Parse the IP Address of the Virtual Machine, Timeout: Error: %s", zap.Error(IPError))
			RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": "Failed to Initialize Virtual Machine"})
			return
		}

		// Getting Initial Configuration for the new Virtual Machine, (only adding with hardware Configuration)
		// All Customer Customization will be added after all.

		NewVirtualMachineConfiguration := models.VirtualMachineConfiguration{

			Metadata: struct {
				VirtualMachineName    string "json:\"VirtualMachineId\" xml:\"VirtualMachineId\""
				VirtualMachineOwnerId string "json:\"VmOwnerId\" xml:\"VmOwnerId\""
			}{
				VirtualMachineName:    VirtualMachineName,
				VirtualMachineOwnerId: strconv.Itoa(CustomerId),
			},

			Datacenter: struct {
				DatacenterName     string `json:"DatacenterName" xml:"DatacenterName"`
				DatacenterItemPath string `json:"DatacenterItemPath" xml:"DatacenterItemPath"`
			}{
				DatacenterName:     Datacenter.Name,
				DatacenterItemPath: object.NewReference(Client.Client, Datacenter.Reference()).(*object.Datacenter).InventoryPath,
			},

			LoadBalancer: struct {
				LoadBalancerPort string "json:\"LoadBalancerPort\" xml:\"LoadBalancerPort\""
				HostMachineIP    string "json:\"HostMachineIP\" xml:\"HostMachineIP\""
			}{
				LoadBalancerPort: "",
				HostMachineIP:    "",
			},

			Network: struct {
				IP       string "json:\"IP,omitempty\" xml:\"IP\""
				Netmask  string "json:\"Netmask,omitempty\" xml:\"Netmask\""
				Hostname string "json:\"Hostname,omitempty\" xml:\"Hostname\""
				Gateway  string "json:\"Gateway,omitempty\" xml:\"Gateway\""
				Enablev6 bool   "json:\"Enablev6,omitempty\" xml:\"Enablev6\""
			}{

				IP:       ParsedResourceInstances["Network"].(*object.Network),
				Hostname: ParsedResourceInstances["Network"].(*mo.Network),
				Enablev6: ParsedResourceInstances["Network"].(*mo.Network),
			},
		}
		// Define Initial ORM Model Object for the Virtual Machine
		NewVirtualMachine := models.VirtualMachine{
			SshInfo:            models.SSHConfiguration{},
			IPAddress:          IPAddress,
			ItemPath:           InitializedInstance.InventoryPath,
			Configuration:      NewVirtualMachineConfiguration,
			OwnerId:            CustomerId,
			VirtualMachineName: VirtualMachineName,
		}

		Created, CreationError := NewVirtualMachine.Create()
		if CreationError != nil {
			Created.Rollback()
			Logger.Error("Failed to Create new ORM VM Object, Error on Creation: %s", zap.Error(CreationError))
		}
		RequestContext.JSON(http.StatusCreated,
			gin.H{"Status": "Initialized"})

	default:
		// In Worse Case returning Initialization Error...
		Logger.Error("Failed to Initialize New Virtual Server, Error: %s", zap.Error(InitError))
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Initialize new Virtual Server"})
	}
}

func DeployVirtualMachineRestController(RequestContext *gin.Context) {

	// Rest Controller, that Applies Configuration on the Existing Initialized Virtual Machine Server
	// Before Calling this Method, you firsly need to call `InitializeVirtualMachineRestController`.

	// Receiving Parsed Configuration of the Characteristics, that has been Provided by User
	// Memory in Megabytes, Cpu Nums etc....

	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(
		RequestContext.Request.Header.Get("Authorization"))

	VmId := RequestContext.Query("VirtualMachineId")
	VmOwnerId := jwtCredentials.UserId

	// Parsing Custom Virtual Machine Configuration
	Deployer := deploy.NewVirtualMachineManager(*Client.Client)
	VmCustomConfig, ParseError := parsers.NewCustomConfig(RequestContext.PostForm("VirtualMachineConfiguration"))
	if ParseError != nil {
		RequestContext.JSON(http.StatusOK, gin.H{"Error": "Invalid Configuration has been Passed"})
		return
	}

	// Receiving Virtual Machine from the Database and Converting into An API Instance...
	VirtualMachine, FindError := Deployer.GetVirtualMachine(VmId, strconv.Itoa(VmOwnerId))

	if FindError != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Server Does Not Exist"})
		return
	}

	// Applying Converted Configuration to the Virtual Machine Instance

	VmInfo, ApplyError := Deployer.ApplyConfiguration(VirtualMachine, *VmCustomConfig)

	switch ApplyError {
	case nil:

		// Updating Virtual Machine ORM Object with New Info

		var VirtualMachine models.VirtualMachine
		var VirtualMachineCustomConfiguration models.VirtualMachineConfiguration
		var VirtualMachineSshConfiguration models.SSHConfiguration

		VirtualMachineSshConfiguration = models.SSHConfiguration{
			Type:               VmInfo.SshType,
			SshCredentialsInfo: VmInfo.SshInfo,
			VirtualMachineId:   VirtualMachine.ID,
		}

		json.Unmarshal(VmCustomConfig.ToJson(), &VirtualMachineCustomConfiguration)

		models.Database.Model(&models.VirtualMachine{}).Where("id = ?").Find(&VirtualMachine)

		VirtualMachine.SshInfo = VirtualMachineSshConfiguration
		VirtualMachine.Configuration = VirtualMachineCustomConfiguration
		VirtualMachine.State = "Ready" // Changing Availability Status To Ready

		RequestContext.JSON(http.StatusOK, gin.H{"Status": "Applied",
			"IPAddress": VmInfo.IPAddress, "SshInfo": VmInfo.SshInfo})

	default:
		Logger.Error("Failed to Apply Configuration to the Virtual Machine, Error: %s", zap.Error(ApplyError))
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": ApplyError})
	}

}

func StartVirtualMachineRestController(RequestContext *gin.Context) {
	// Rest Controller, that is Used to Start Virtual Machine Server

	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(RequestContext.Request.Header.Get("Authorization"))
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := jwtCredentials.UserId

	// Updating the State of the Virtual Machine to `NotReady` in order to prevent other operations on this Virtual Machine

	var VirtualMachine models.VirtualMachine
	models.Database.Model(&models.VirtualMachine{}).Where(
		"id = ?", VirtualMachineId).Find(&VirtualMachine)

	VmManager := deploy.NewVirtualMachineManager(*Client.Client)
	Vm, VmError := VmManager.GetVirtualMachine(VirtualMachineId, strconv.Itoa(CustomerId))

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

	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(RequestContext.Request.Header.Get("Authorization"))
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := jwtCredentials.UserId
	NewVmManager := deploy.NewVirtualMachineManager(*Client.Client)
	Vm, VmError := NewVmManager.GetVirtualMachine(VirtualMachineId, strconv.Itoa(CustomerId))

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

	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(
		RequestContext.Request.Header.Get("jwt-token"))

	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := jwtCredentials.UserId

	NewVmManager := deploy.NewVirtualMachineManager(*Client.Client)
	Vm, FindError := NewVmManager.GetVirtualMachine(VirtualMachineId, strconv.Itoa(CustomerId))

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

		var VirtualMachine models.VirtualMachine
		models.Database.Model(&models.VirtualMachine{}).Where(

			"id = ?", VirtualMachineId).Find(&VirtualMachine)
		Deleted, Error := VirtualMachine.Delete()

		if Error != nil {
			Deleted.Rollback()
			Logger.Error(
				"Failed to Delete Virtual Machine Object with ID: `%s`, Error: %s",
				zap.String("Virtual Machine ID", VirtualMachineId), zap.Error(Error))
		}
		RequestContext.JSON(http.StatusCreated, gin.H{"Operation": "Success"})
	}
}

func RebootGuestOSRestController(RequestContext *gin.Context) {
	// Rest Controller, that allows to Reboot Operational System of the Virtual Machine
	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(RequestContext.Request.Header.Get("jwt-token"))
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	VmOwnerId := jwtCredentials.UserId

	VmManager := deploy.NewVirtualMachineManager(*Client.Client)
	VirtualMachine, FindError := VmManager.GetVirtualMachine(VirtualMachineId, strconv.Itoa(VmOwnerId))
	if FindError != nil {
		RequestContext.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Machine Server not found"})
		return
	}
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*20)
	defer CancelFunc()

	RebootedError := VirtualMachine.RebootGuest(TimeoutContext)
	if RebootedError != nil {
		Logger.Error("Failed to Reboot OS on Virtual Machine Server, Error: %s", zap.Error(RebootedError))
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": "Failed to Reboot Operational System"})
		return
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Status": "Rebooted"})
}

func StartGuestOSRestController(RequestContext *gin.Context) {
	// Rest Controller, that allows to Start Operational System on the Virtual machine

	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(RequestContext.Request.Header.Get("Authorization"))
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	VmOwnerId := jwtCredentials.UserId

	VmManager := deploy.NewVirtualMachineManager(*Client.Client)
	VirtualMachine, FindError := VmManager.GetVirtualMachine(VirtualMachineId, strconv.Itoa(VmOwnerId))
	if FindError != nil {
		RequestContext.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Machine Server not found"})
		return
	}
	// Starting Virtual Machine...
	RebootedError := VmManager.StartVirtualMachine(VirtualMachine)
	if RebootedError != nil {
		Logger.Error("Failed to Reboot OS on Virtual Machine Server, Error: %s", zap.Error(RebootedError))
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": "Failed to Reboot Operational System"})
		return
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Status": "Started"})
}

func ShutdownGuestOsRestController(RequestContext *gin.Context) {
	// Rest Controller, that allows to Shutdown Operational System on the Virtual Machine
	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(RequestContext.Request.Header.Get("Authorization"))
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	VmOwnerId := jwtCredentials.UserId

	VmManager := deploy.NewVirtualMachineManager(*Client.Client)
	VirtualMachine, FindError := VmManager.GetVirtualMachine(VirtualMachineId, strconv.Itoa(VmOwnerId))
	if FindError != nil {
		RequestContext.AbortWithStatusJSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Machine Server not found"})
		return
	}
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*20)
	defer CancelFunc()

	RebootedError := VirtualMachine.ShutdownGuest(TimeoutContext)
	if RebootedError != nil {
		Logger.Error("Failed to Shutdown OS on Virtual Machine Server, Error: %s", zap.Error(RebootedError))
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": "Failed to shutdown Operational System"})
		return
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Status": "Shutdowned"})
}
