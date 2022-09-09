package suggestion_rest

import (
	"context"
	_ "context"
	"encoding/json"

	"net/http"
	"net/url"

	"os"
	"time"
	_ "time"

	"github.com/LovePelmeni/Infrastructure/host_system"
	"github.com/LovePelmeni/Infrastructure/resources"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/rest"
	_ "github.com/vmware/govmomi/vapi/rest"
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
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("SuggestionsRestLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {

	InitializeProductionLogger()
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
			Logger.Error("FAILED TO LOGIN TO THE VMWARE HYPERVISOR SERVER, ERROR: %s", zap.Error(FailedToLogin))
		}
	}
	Client = APIClient
}

// Suggestions Resources API Controllers

func GetDatacentersSuggestions(RequestContext *gin.Context) {
	// Returns Suggestions for the Datacenter Resource, based on the Customer Needs

	// Initializing Resource Requirements
	Requirements := RequestContext.PostForm("ResourceRequirements")
	ResourceRequirements, Error := resources.NewDatacenterResourceRequirements(Requirements)

	// If Failed to Find Any Available Datacenters, due to the Error, Returning Bad Request with Error Explanation
	if Error != nil {
		Logger.Error("Failed to Parse Query Of Available Datacenters", zap.Error(Error))
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

func GetAvailableInstallationToolsRestController(RequestContext *gin.Context) {
	// Return Array of the Tools, that can be pre-installed on the Virtual Machine Server
	Tools := []struct {
		ToolName string `json:"ToolName"`
	}{
		{ToolName: "Docker"},
		{ToolName: "Docker-Compose"},
		{ToolName: "Podman"},
		{ToolName: "VirtualBox"},
	}
	SerializedTools, Error := json.Marshal(Tools)
	if Error != nil {
		RequestContext.JSON(http.StatusOK, gin.H{"Error": Error})
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Tools": SerializedTools})
}

func GetAvailableLoadBalancersRestController(Request *gin.Context) {
	// Returns array of the Available Load Balancers
	LoadBalancers := []struct {
		LoadBalancerName string `json:"LoadBalancerName" xml:"LoadBalancerName"`
	}{
		{LoadBalancerName: "nginx"},
		{LoadBalancerName: "apache"},
	}
	Request.JSON(http.StatusOK, gin.H{"QuerySet": LoadBalancers})
}
