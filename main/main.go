package main

import (
	"context"
	"errors"
	"log"

	"fmt"
	"net/http"

	"os"
	"os/signal"
	"syscall"

	"github.com/LovePelmeni/Infrastructure/healthcheck_rest"
	"github.com/LovePelmeni/Infrastructure/middlewares"
	"github.com/LovePelmeni/Infrastructure/ssh_rest"

	customer_rest "github.com/LovePelmeni/Infrastructure/customer_rest"
	load_balancer_rest "github.com/LovePelmeni/Infrastructure/load_balancer_rest"
	suggestion_rest "github.com/LovePelmeni/Infrastructure/suggestion_rest"
	vm_rest "github.com/LovePelmeni/Infrastructure/vm_rest"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	APPLICATION_HOST = os.Getenv("APPLICATION_HOST")
	APPLICATION_PORT = os.Getenv("APPLICATION_PORT")

	FRONT_APPLICATION_HOST = os.Getenv("FRONT_APPLICATION_HOST")
	FRONT_APPLICATION_PORT = os.Getenv("FRONT_APPLICATION_PORT")
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("Main.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

type Server struct {
	ServerHost string `json:"ServerHost"`
	ServerPort string `json:"ServerPort"`
}

func NewServer(ServerHost string, ServerPort string) *Server {
	return &Server{
		ServerHost: ServerHost,
		ServerPort: ServerPort,
	}
}

func (this *Server) Run() {

	Router := gin.Default()
	// Setting Up Cross Origin Resource Sharing Policy

	Router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			fmt.Sprintf("http://%s:%s", APPLICATION_HOST, APPLICATION_PORT),
			fmt.Sprintf("http://%s:%s", FRONT_APPLICATION_HOST, FRONT_APPLICATION_PORT),
		},
		AllowMethods:     []string{"POST", "PUT", "DELETE", "GET", "OPTIONS"},
		AllowCredentials: true,
		AllowHeaders:     []string{"*"},
		AllowWebSockets:  false,
	}))

	// Setting up Healthcheck Rest Endpoint

	Router.GET("/ping/", func(context *gin.Context) {
		context.JSON(http.StatusOK, nil)
	})

	// Customers Rest API Endpoints

	CustomerGroup := Router.Group("/customer/")
	{
		CustomerGroup.POST("/login/", customer_rest.LoginRestController, middlewares.NonAuthorizationRequiredMiddleware())
		CustomerGroup.POST("/logout/", customer_rest.LogoutRestController, middlewares.AuthorizationRequiredMiddleware())

		CustomerGroup.POST("/create/", customer_rest.CreateCustomerRestController, middlewares.NonAuthorizationRequiredMiddleware())
		CustomerGroup.PUT("/reset/password/", customer_rest.ResetPasswordRestController, middlewares.AuthorizationRequiredMiddleware())
		CustomerGroup.DELETE("/delete/", customer_rest.DeleteCustomerRestController, middlewares.AuthorizationRequiredMiddleware())
		CustomerGroup.GET("/get/profile/", customer_rest.GetCustomerProfileRestController, middlewares.AuthorizationRequiredMiddleware())
	}

	// Load Balancers Rest API Endpoints

	LoadBalancerGroup := Router.Group("/route/").Use(
		middlewares.AuthorizationRequiredMiddleware(),
		middlewares.InfrastructureHealthCircuitBreakerMiddleware(),
	)
	{
		LoadBalancerGroup.POST("/create/", load_balancer_rest.AddWebserverVirtualMachineRouteRestController)
		LoadBalancerGroup.DELETE("/delete/", load_balancer_rest.RemoveWebServerVirtualMachineRouteRestController)
	}

	// Virtual Machines Rest API Endpoints
	VirtualMachineGroup := Router.Group("/vm/").Use(
		middlewares.AuthorizationRequiredMiddleware(),
		middlewares.IsVirtualMachineOwnerMiddleware(),
		middlewares.InfrastructureHealthCircuitBreakerMiddleware(),
		middlewares.IsReadyToPerformOperationMiddleware())
	{
		{
			VirtualMachineGroup.POST("/initialize/", vm_rest.InitializeVirtualMachineRestController) // initialized new Virtual Machine (Emtpy)
			VirtualMachineGroup.PUT("/deploy/", vm_rest.DeployVirtualMachineRestController)          // Applies Configuration to the Initialized Machine
			VirtualMachineGroup.DELETE("/remove/", vm_rest.RemoveVirtualMachineRestController)       // Removes Existing Virtual Machine
			VirtualMachineGroup.POST("/start/", vm_rest.StartVirtualMachineRestController)           // Starts Virtual Machine
			VirtualMachineGroup.POST("/reboot/", vm_rest.RebootVirtualMachineRestController)         // Reboots Virtual Machine
			VirtualMachineGroup.DELETE("/shutdown/", vm_rest.ShutdownVirtualMachineRestController)   // Shutting Down Virtual Machine
		}

		{
			VirtualMachineGroup.GET("/get/list/", vm_rest.GetCustomerVirtualMachine) // Customer's Virtual Machines
			VirtualMachineGroup.GET("/get/", vm_rest.GetCustomerVirtualMachines)     // Customer's Specific Virtual Machine
		}
		VirtualMachineGroup.GET("/health/metrics/", healthcheck_rest.GetVirtualMachineHealthMetricRestController) // HealthCheck Metrics of the Virtual Machine
	}

	// Host System Rest Endpoints
	HostSystemGroup := Router.Group("/host/").Use(

		middlewares.AuthorizationRequiredMiddleware(),
		middlewares.IsVirtualMachineOwnerMiddleware(),
		middlewares.InfrastructureHealthCircuitBreakerMiddleware(),
		middlewares.IsReadyToPerformOperationMiddleware())
	{
		HostSystemGroup.POST("system/start/", vm_rest.StartGuestOSRestController)
		HostSystemGroup.PUT("system/restart/", vm_rest.RebootGuestOSRestController)
		HostSystemGroup.DELETE("system/shutdown/", vm_rest.ShutdownGuestOsRestController)
	}

	// SSH Rest Endpoints

	SshSystemGroup := Router.Group("/ssh/").Use(

		middlewares.AuthorizationRequiredMiddleware(),
		middlewares.IsVirtualMachineOwnerMiddleware(),
		middlewares.InfrastructureHealthCircuitBreakerMiddleware(),
		middlewares.IsReadyToPerformOperationMiddleware(),
	)
	{
		SshSystemGroup.PUT("/recover/", ssh_rest.UpdateVirtualMachineSshKeysRestController)
	}

	// Suggestions Rest Endpoints
	SuggestionsGroup := Router.Group("/suggestions/").Use(
		middlewares.AuthorizationRequiredMiddleware(),
		middlewares.InfrastructureHealthCircuitBreakerMiddleware())
	{
		SuggestionsGroup.POST("/datacenter/", suggestion_rest.GetDatacentersSuggestions) // do not change to Safe Methods such as GET, OPTIONS, etc...
		SuggestionsGroup.GET("/os/", suggestion_rest.GetAvailableOsSystemsRestController)
		SuggestionsGroup.GET("/load/balancer/", suggestion_rest.GetAvailableLoadBalancersRestController)
		SuggestionsGroup.GET("/pre/installed/tool/", suggestion_rest.GetAvailableInstallationToolsRestController)
	}

	// Support Rest API Endpoints

	SupportGroup := Router.Group("/support/").Use(middlewares.AuthorizationRequiredMiddleware())
	{
		SupportGroup.POST("/feedback/", customer_rest.SupportRestController)
	}

	Server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", this.ServerHost, this.ServerPort),
		Handler: Router,
	}

	ServerShutDownContext, ErrorCancelMethod := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT) // Creating Notify Context that triggers server to shut down
	// after receiving system SIGTERM or SIGQUIT Signal from Kubernetes / Localhost.

	go this.Shutdown(ServerShutDownContext, ErrorCancelMethod, Server)

	Exception := Server.ListenAndServe()
	if errors.Is(Exception, http.ErrServerClosed) {
		ServerShutDownContext.Done()
	} else {
		fmt.Print("Server has been Shutdown For Some Reason, Check `Main.log` for more info")
		ErrorLogger.Printf(
			"Error while Running the Server: %s", Exception.Error())
		ServerShutDownContext.Done()
	}
}

func (this *Server) Shutdown(Context context.Context, CancelFunc context.CancelFunc, ServerInstance *http.Server) {
	select {
	case <-Context.Done():
		defer CancelFunc()
		ShutdownError := ServerInstance.Shutdown(context.Background())
		DebugLogger.Printf("Server has been Shutdown, Errors: %s", ShutdownError)
	}
}

func main() {
	DebugLogger.Printf("Running Http Application Server...")
	httpServer := NewServer(APPLICATION_HOST, APPLICATION_PORT)
	httpServer.Run()
}
