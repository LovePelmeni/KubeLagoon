package resources_rest

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {

	LogFile, Error := os.OpenFile("RestResources.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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

func GetClusterComputeResourceSuggestions(RequestContext *gin.Context) {
	// Returns Suggestions for the Cluster Compute Resource, based on the Customer Needs

	CPUNums := RequestContext.PostForm("CpuNums")
	MemoryInMegabytes := RequestContext.PostForm("MemoryInMegabytes")
	ResourceRequirements := suggestions.NewResourceRequirements(map[string]any{
		"CPUNums": CPUNums,
		"Memory":  MemoryInMegabytes,
	})

	SuggestionClusterManager := suggestions.NewClusterComputeResourceSuggestManager()
	SuggestedResources, SuggestError := SuggestionClusterManager.GetSuggestions(*ResourceRequirements)
	switch SuggestError {
	case nil:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": SuggestedResources})
		return
	default:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": "No Cluster Compute Resources is Available"})
	}
}

func GetDatastoreSuggestions(RequestContext *gin.Context) {
	// Returns Suggestions for the Datastore Resource, based on the Customer Needs
	MemoryInMegabytes := RequestContext.PostForm("MemoryInMegabytes")
	ResourceRequirements := suggestions.NewResourceRequirements(map[string]any{
		"Memory": MemoryInMegabytes,
	})

	SuggestionClusterManager := suggestions.NewDatastoreSuggestManager(*Client.Client)
	SuggestedResources := SuggestionClusterManager.GetSuggestions(*ResourceRequirements)
	switch len(SuggestedResources) {
	default:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": SuggestedResources})
		return
	case 0:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": "No Cluster Compute Resources is Available"})
	}

}

func GetStorageSuggestions(RequestContext *gin.Context) {
	// Returns Suggestions for the Storage Resource, based on the Customer Needs

	CapacityInGB, _ := strconv.Atoi(RequestContext.PostForm("Capacity"))
	ResourceRequirements := suggestions.NewResourceRequirements(map[string]any{
		"Capacity": CapacityInGB * 1024 * 1024,
	})

	SuggestionClusterManager := suggestions.NewStorageSuggestManager(*Client.Client)
	SuggestedResources := SuggestionClusterManager.GetSuggestions(*ResourceRequirements)
	switch len(SuggestedResources) {
	case nil:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": SuggestedResources})
		return
	default:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": "No Storages is Available"})
	}
}

func GetDatacenterSuggestions(RequestContext *gin.Context) {
	// Returns Suggestions for the Datacenter Resource, based on the Customer Needs

	ResourceRequirements := suggestions.NewResourceRequirements()
	SuggestionClusterManager := suggestions.NewDataCenterSuggestManager(*Client.Client)
	SuggestedResources := SuggestionClusterManager.GetSuggestions(*ResourceRequirements)
	switch len(SuggestedResources) {
	default:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": SuggestedResources})
		return
	case 0:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": "No Datacenters is Available"})
	}

}

func GetFolderSuggestions(RequestContext *gin.Context) {
	// Returns Suggestions for the Folder Resource, based on the Customer Needs

	Requirements := suggestions.NewResourceRequirements()
	SuggestionClusterManager := suggestions.NewFolderSuggestManager(*Client.Client)
	SuggestedResources := SuggestionClusterManager.GetSuggestions(*Requirements)
	switch len(SuggestedResources) {
	default:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": SuggestedResources})
		return
	case 0:
		RequestContext.JSON(http.StatusOK,
			gin.H{"QuerySet": "No Folders is Available"})
	}

}
