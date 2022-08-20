package rest

import (
	"context"
	"encoding/json"
	"log"
	"reflect"

	"fmt"
	"sync"

	_ "net"
	"net/http"

	_ "net/smtp"
	"net/url"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/LovePelmeni/Infrastructure/parsers"
	"github.com/LovePelmeni/Infrastructure/suggestions"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/rest"

	"github.com/vmware/govmomi/vim25"
	_ "github.com/xhit/go-simple-mail/v2"
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
	SUPPORT_EMAIL_ADDRESS  = os.Getenv("SUPPORT_EMAIL_ADDRESS")
	SUPPORT_EMAIL_PASSWORD = os.Getenv("SUPPORT_EMAIL_PASSWORD")

	SUPPORT_CLIENT_EMAIL_ADDRESS  = os.Getenv("SUPPORT_CLIENT_EMAIL_ADDRESS")
	SUPPORT_CLIENT_EMAIL_PASSWORD = os.Getenv("SUPPORT_CLIENT_EMAIL_PASSWORD")
)

var (
	Client govmomi.Client
)

var (
	VirtualMachine models.VirtualMachine
	configuration  models.Configuration
	Customer       models.Customer
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {

	LogFile, Error := os.OpenFile("Rest.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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

// Authorization Rest API Endpoints

func LoginRestController(RequestContext *gin.Context) {
	// Rest Controller, that is responsible for users to let them login

	Username := RequestContext.PostForm("Username")
	Password := RequestContext.PostForm("Password")

	var Customer models.Customer
	customer := models.Database.Model(&models.Customer{}).Where(
		"username = ?", Username).Find(&Customer)

	if customer.Error != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "No User With This Username Exists :("})
		return
	}

	if EqualsError := bcrypt.CompareHashAndPassword(
		[]byte(Customer.Password), []byte(Password)); EqualsError != nil {
		RequestContext.JSON(http.StatusBadRequest, gin.H{"Error": "Invalid Password"})
	}
}

func LogoutRestController(RequestContext *gin.Context) {
	// Rest Controller, that is responsible to let users Log out from their existing account

	if Cookie, Error := RequestContext.Cookie("jwt-token"); len(Cookie) != 0 && Error == nil {
		Cookie, _ := RequestContext.Request.Cookie("jwt-token")
		Cookie.Expires.Add(-1)
		RequestContext.JSON(http.StatusOK, gin.H{"Status": "Logged out"})
	}
}

// Customers Rest API Endpoints

func CreateCustomerRestController(RequestContext *gin.Context) {

	NewCustomer := models.NewCustomer()
	Created, Error := NewCustomer.Create()

	if reflect.ValueOf(Created).IsNil() || Error != nil {

		switch Error {
		case gorm.ErrInvalidData:
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "Invalid Credentials has been Passed, Make sure that Credentials has proper Length and Character Type"})
			return

		case gorm.ErrInvalidValue:
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "Invalid Value has been Passed"})
			return

		case gorm.ErrModelValueRequired:
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": "You missed to Setup Required Fields"})
			return

		default:
			RequestContext.JSON(http.StatusBadRequest,
				gin.H{"Error": fmt.Sprintf("Unknown Error `%s`", Error.Error())})
			return
		}
	} else {
		RequestContext.JSON(http.StatusCreated, gin.H{"Operation": "Success"})
	}
}

func UpdateCustomerRestController(RequestContext *gin.Context) {

}

func DeleteCustomerRestController(RequestContext *gin.Context) {
	CustomerId := RequestContext.Query("CustomerId")
	var Customer models.Customer
	models.Database.Model(&models.Customer{}).Where("id = ?", CustomerId).Find(&Customer)
	_, Error := Customer.Delete()

	switch Error {
	case gorm.ErrRecordNotFound:
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Profile with this Credentials Does Not Exist"})

	case gorm.ErrInvalidTransaction:
		ErrorLogger.Printf(
			"Failed to Delete Customer Profile with ID: %s, Error: %s", CustomerId, Error)
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Delete Profile"})
	default:
		ErrorLogger.Printf("Unknown Error on Customer Deletion, Error: %s", Error)
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": fmt.Sprintf("Unknown Error, %s", Error)})
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
	VirtualMachineConfig, _ := models.NewConfiguration(*HardwareConfiguration, *CustomConfiguration).Create()
	VirtualMachine, _ := models.NewVirtualMachine(CustomerId,
		CustomConfiguration.Metadata.VirtualMachineName, DeployedInstance.InventoryPath).Create()

	switch DeployError {

	case nil:

		VirtualMachineConfig.Commit()
		VirtualMachine.Commit()

		RequestContext.JSON(http.StatusOK,
			gin.H{"Operation": "Success"})
	default:

		VirtualMachineConfig.Rollback()
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

// Resources Rest API Endpoints

func GetAvailableResourcesInfoRestController(context *gin.Context) {

	// Rest Endpoint, Returns All Available Resources, to Configure the Virtual Machine Server

	ResourceTypes := map[string]suggestions.SuggestManagerInterface{
		"DataCenters": suggestions.NewDataCenterSuggestManager(*Client.Client),
		"DataStores":  suggestions.NewDatastoreSuggestManager(*Client.Client),
		"Networks":    suggestions.NewNetworkSuggestManager(*Client.Client),
		"Resources":   suggestions.NewResourceSuggestManager(*Client.Client),
		"Folders":     suggestions.NewFolderSuggestManager(*Client.Client),
	}

	var Resources map[string][]suggestions.ResourceSuggestion
	for ResourceName, ResourceManager := range ResourceTypes {
		Resources[ResourceName] = ResourceManager.GetSuggestions()
	}
	SerializedResources, SerializeError := json.Marshal(Resources)
	switch {
	case SerializeError != nil: // If Failed to Serialize Resource Suggestions
		context.JSON(http.StatusOK, gin.H{"Resources": Resources})

	case SerializeError == nil:
		context.JSON(http.StatusOK, gin.H{"Resources": SerializedResources})

	default:
		context.JSON(http.StatusOK, gin.H{"Resources": SerializedResources})
	}
}

// Virtual Machine INFO Rest API Endpoints

func GetCustomerVirtualMachinesRestController(RequestContext *gin.Context) {

	// Rest Controller, that returns Info about all Virtual Machines, that Customer
	// Have, Including Current health, SshCredentials, Status, CPU/Memory etc....

	CustomerId := RequestContext.Query("customerId")
	var Queryset []struct {
		Vm     models.VirtualMachine
		Status string
	}

	var CustomerVirtualMachines []models.VirtualMachine

	models.Database.Model(&models.VirtualMachine{}).Where(
		"owner_id = ?", CustomerId).Preload("Vms").Find(&CustomerVirtualMachines)

	group := sync.WaitGroup{}

	for _, VirtualMachine := range CustomerVirtualMachines {

		go func() {
			group.Add(1)
			TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*20)
			defer CancelFunc()

			Finder := object.NewSearchIndex(&vim25.Client{})
			VmEntity, _ := Finder.FindByInventoryPath(TimeoutContext, VirtualMachine.ItemPath)
			PowerState, _ := VmEntity.(*object.VirtualMachine).PowerState(TimeoutContext)
			VirtualMachineQuerySet = struct {
				VirtualMachineName string `json"VirtualMachineName"`
				Status string `json:"Status"`
			}{
				VirtualMachineName: VirtualMachine.VirtualMachineName, 
				Status: string(PowerState),
			}
			Queryset = append(Queryset, VirtualMachineQuerySet)
			group.Done()
		}()
		group.Wait()
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Queryset": Queryset})
}

func GetCustomerVirtualMachineInfoRestController(RequestContext *gin.Context) {
	// Rest Controller, that returns Info about Specific Customer Virtual Machine,
	// Including Current Health, Status, Ssh Credentials, CPU/memory usage, etc...


	type Config struct {

		Metadata struct {
			FullName string `json:"FullName"`
			Version string `json:"Version"`
			InstanceUuid string `json:"InstanceUuid"`
			Vendor string `json:"Vendor"`
		} `json:"Metadata"`

		PowerState string `json:"PowerState"`

		IPInfo struct {
			Tip string `json:"ChapterTip"`
			IPAddress string `json:"IPAddress"`
		} `json:"IpInfo" xml:"IPInfo"`

		TlsInfo struct {
			Tip string `json:"ChapterTip"`
			PrivateKey []byte  `json:"PrivateKey"`
			PublicKey []byte `json:"PublicKey"`
		} `json:"TlsInfo" xml:"TlsInfo"` 

		OSInfo struct {
			OSName string `json:"OSName"`
			Bit    int    `json:"Bit;omitempty;"`
			Version string `json:"OSVersion;omitempty;"`
		}

		DiskInfo struct {
			Tip string `json:"ChapterTip"`
			Capacity string `json:"Capacity"`
			DiskType string `json:"DiskType"`
		} `json:"DiskInfo" xml:"DiskInfo"`

		NetworkInfo struct {
			Tip string `json:"ChapterTip"`
			Netmask string `json:"`
			NetworkIP string `json:"`
		} `json:"NetworkInfo" xml:"NetworkInfo"`

		DatacenterInfo struct {
			Name string `json:"Name"`
			UniqueName string `json:"UniqueName"`
		} `json:"Datacenter"`

		DatastoreInfo struct {
			Name string `json:"Name"`
			UniqueName string `json:"UniqueName"`
			MemoryInUse int32 `json:"MemoryInUse"`
		} `json:"Datastore`
	}

	CustomerId := RequestContext.Query("customerId")
	VirtualMachineId := RequestContext.Query("virtualMachineId")

	var VirtualMachine models.VirtualMachine

	models.Database.Model(&models.Customer{}).Where(
		"id = ?", CustomerId).Preload("Vms").Where("id = ?",
		VirtualMachineId).Find(&VirtualMachine)

	group := sync.WaitGroup{}

	go func() { // Receiving the State of the Virtual Machine
		group.Add(1)
		Timeout, CancelFunc := context.WithTimeout(context.Background(), time.Second*20)
		defer CancelFunc()

		Finder := object.NewSearchIndex(&vim25.Client{})
		Vm, _ := Finder.FindByInventoryPath(Timeout, VirtualMachine.ItemPath)
		PowerState, _ := Vm.(*object.VirtualMachine).PowerState(Timeout)

		VmServiceContent := Vm.(*object.VirtualMachine).Client().ServiceContent

		VirtualMachineQuerySet := Config{
			Metadata: struct {
				FullName string `json:"FullName"`
				Version  string `json:"Version"`
				InstanceUuid string `json:"InstanceUuid"`
				Vendor string `json:"Vendor"`
			}{
				FullName: VmServiceContent.About.FullName, 
				InstanceUuid: VmServiceContent.About.InstanceUuid, 	
				Version: VmServiceContent.About.Version, 
				Vendor: VmServiceContent.About.Vendor,
			},
			PowerState: string(PowerState),
			IPInfo: struct {
				Tip string `json:"ChapterTip"`
				IPAddress string `json:"IPAddress"`
			}{
			Tip: "This Is IP Info about your Virtual Server," +
			" there you can find Info about Location of your Virtual Server", 
			IPAddress: VmServiceContent.IpPoolManager.Value,
			},
			OSInfo: struct {
				OSName string "json:\"OSName\"";
				Bit int "json:\"Bit;omitempty;\"";
				Version string "json:\"OSVersion;omitempty;\""
			}{
				OSName: Vm.(*object.VirtualMachine).Client().ServiceContent.About.OsType, 

			},

			TlsInfo: struct{
				Tip string "json:\"ChapterTip\"";
				PrivateKey []byte "json:\"PrivateKey\""
				PublicKey []byte "json:\"PublicKey\""
			}{
				Tip: "This Is TLS/SSL Configuration for your Virtual Server, to Access your VM Server Using SSH run: `ssh user@password -i <public-key>", 
				PrivateKey: Vm.(*object.VirtualMachine).Client().Certificate().PrivateKey, 
				PublicKey: Vm.(*object.VirtualMachine).Client().Certificate().Leaf.PublicKey, 
			},
			DiskInfo: struct{
			Tip string "json:\"ChapterTip\"";
			 Capacity string "json:\"Capacity\""; 
			 DiskType string "json:\"DiskType\""
			 RootFolder string "json:\"RootFolder\""
			}{
				RootFolder: Vm.(*object.VirtualMachine).Client().ServiceContent.RootFolder.Value,
				DiskType: VmServiceContent.VirtualDiskManager.Type,
				Capacity: VirtualMachine
			},
			
		
		}

		group.Done()
	}()

	group.Wait()

	Vm := struct {
		VirtualMachine models.VirtualMachine
		Status         string
	}{
		VirtualMachine: VirtualMachine,
		Status:         string(PowerState),
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Vm": Vm})
}

func SupportRestController(RequestContext *gin.Context) {
	// Rest Controller, that is used for sending Email Support Notifications
}
