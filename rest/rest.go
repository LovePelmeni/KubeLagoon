package rest

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/rest"
)

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

func SupportRestController(context *gin.Context) {

}
