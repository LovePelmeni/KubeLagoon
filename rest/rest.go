package rest

import (
	"context"
	"encoding/json"
	"sync"

	_ "net"
	"net/http"

	_ "net/smtp"
	"net/url"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/models"

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
	APIIp    = os.Getenv("API_SOURCE_IP")
	Username = os.Getenv("API_SOURCE_USERNAME")
	Password = os.Getenv("API_SOURCE_PASSWORD")

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

func init() {

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
}

// Authorization Rest API Endpoints

func LoginRestController(RequestContext *gin.Context) {
}

func LogoutRestController(RequestContext *gin.Context) {
}

// Customers Rest API Endpoints

func CreateCustomerRestController(RequestContext *gin.Context) {

}

func UpdateCustomerRestController(RequestContext *gin.Context) {

}

func DeleteCustomerRestController(RequestContext *gin.Context) {

}

// Virtual Machine Rest API Endpoints

func DeployNewVirtualMachineRestController(RequestContext *gin.Context) {
}

func UpdateVirtualMachineConfigurationRestController(RequestContext *gin.Context) {
}

func ShutdownVirtualMachineRestController(RequestContext *gin.Context) {

}

func RemoveVirtualMachineRestController(context *gin.Context) {

}

// Support Rest API Endpoints

func SupportRestController(context *gin.Context) {
}

// Resources Rest API Endpoints

func GetAvailableResourcesInfoRestController(context *gin.Context) {

	// Rest Endpoint, Returns All Available Resources, to Configure the Virtual Machine Server

	ResourceTypes := map[string]suggestions.SuggestManagerInterface{
		"DataCenters": suggestions.NewDataCenterSuggestManager(),
		"DataStores":  suggestions.NewDatastoreSuggestManager(),
		"Networks":    suggestions.NewNetworkSuggestManager(),
		"Resources":   suggestions.NewResourceSuggestManager(),
		"Folders":     suggestions.NewFolderSuggestManager(),
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

			VirtualMachineQuerySet := struct {
				Vm     models.VirtualMachine
				Status string
			}{
				Vm:     VirtualMachine,
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

	CustomerId := RequestContext.Query("customerId")
	VirtualMachineId := RequestContext.Query("virtualMachineId")

	var VirtualMachine models.VirtualMachine
	var PowerState string

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
		Pw, _ := Vm.(*object.VirtualMachine).PowerState(Timeout)
		PowerState = string(Pw)
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
