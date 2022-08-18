package rest

import (
	"context"
	"encoding/json"
	_ "net"
	"net/http"
	_ "net/smtp"
	"net/url"
	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/parsers"
	"github.com/LovePelmeni/Infrastructure/suggestions"
	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/rest"
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

	ConfigurationParser := parsers.NewConfigurationParser()
	VirtualMachineDeployer := deploy.NewVirtualMachineDeployer()
	ParsedConfiguration, ParseError := ConfigurationParser.ConfigParse(
		[]byte(RequestContext.PostForm("NewConfiguration")))

	switch {
	case ParseError == nil:
		InitializedMachine, InitializeError := VirtualMachineDeployer.InitializeNewVirtualMachine()
		DeployedVm := VirtualMachineDeployer.StartVirtualMachine()
	}
}

func UpdateVirtualMachineConfigurationRestController(RequestContext *gin.Context) {

	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	CustomerId := RequestContext.Query("CustomerId")

	NewVirtualMachineDeployer := deploy.NewVirtualMachineDeployer()
	NewConfigurationParser := parsers.NewConfigurationParser()

	ParsedConfiguration, ParserError := NewConfigurationParser.ConfigParse([]byte(RequestContext.PostForm("Configuration")))
	VirtualMachine, NotExistsError := NewVirtualMachineDeployer.GetVirtualMachine(RequestContext.Query("VirtualMachineId"), RequestContext.Query("CustomerId"))

	if ParserError == nil {

		go func() {
			group.Add(1)
			InitializedMachine, InitializeError := NewVirtualMachineDeployer.InitializeNewVirtualMachine()
			DeployedError := NewVirtualMachineDeployer.StartVirtualMachine(InitializedMachine)
			group.Done()
		}()

	}
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

func GetCustomerVirtualMachinesRestController(context *gin.Context) {
	// Rest Controller, that returns Info about all Virtual Machines, that Customer
	// Have, Including Current health, SshCredentials, Status, CPU/Memory etc....
}

func GetCustomerVirtualMachineInfoRestController(context *gin.Context) {
	// Rest Controller, that returns Info about Specific Customer Virtual Machine,
	// Including Current Health, Status, Ssh Credentials, CPU/memory usage, etc...
}
