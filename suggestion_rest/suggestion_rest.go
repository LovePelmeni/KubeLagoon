package suggestion_rest

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/host_system"
	"github.com/LovePelmeni/Infrastructure/resources"
	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
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
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {

	LogFile, Error := os.OpenFile("../logs/RestResources.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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

// Suggestions Resources API Controllers

func GetDatacenterSuggestions(RequestContext *gin.Context) {
	// Returns Suggestions for the Datacenter Resource, based on the Customer Needs

	// Initializing Resource Requirements
	Requirements := RequestContext.PostForm("ResourceRequirements")
	ResourceRequirements, Error := resources.NewDatacenterResourceRequirements(Requirements)

	// If Failed to Find Any Available Datacenters, due to the Error, Returning Bad Request with Error Explanation
	if Error != nil {
		ErrorLogger.Printf("Failed to Parse Query Of Available Datacenters, Error: %s", Error)
		RequestContext.JSON(http.StatusBadGateway, gin.H{"Error": Error})
	}

	// Receiving the QuerySet of the Available Datacenters, according to the Resource Requirements
	SuggestionDatacenterManager := resources.NewDatacenterResourceManager(Client.Client)
	SuggestedResources := SuggestionDatacenterManager.GetAvailableDatacenters(*ResourceRequirements)

	switch len(SuggestedResources) {
	default:
		RequestContext.JSON(http.StatusOK,
			gin.H{"Datacenters": SuggestedResources})
	case 0:
		RequestContext.JSON(http.StatusOK,
			gin.H{"Datacenters": make([]*object.Datacenter, 0)})
	}
}

func GetAvailableOsSystemsRestController(RequestContext *gin.Context) {
	// Returns List of Available System Distributions for Linux and Windows

	HostSystemManager := host_system.NewVirtualMachineHostSystemManager()
	WindowsHostSystems := HostSystemManager.GetAvailableWindowsOsSystems()
	LinuxHostSystems := HostSystemManager.GetAvailableLinuxOsSystems()
	RequestContext.JSON(http.StatusOK, gin.H{"Linux": LinuxHostSystems,
		"Windows": WindowsHostSystems})
}
