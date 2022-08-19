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

	"github.com/LovePelmeni/Infrastructure/middlewares"
	"github.com/LovePelmeni/Infrastructure/rest"

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
	httpServer := http.Server{
		Addr:    fmt.Sprintf("%s:%s", this.ServerHost, this.ServerPort),
		Handler: Router,
	}

	// Setting Up Cross Origin Resource Sharing Policy

	Router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			fmt.Sprintf("%s:%s", APPLICATION_HOST, APPLICATION_PORT),
			fmt.Sprintf("%s:%s", FRONT_APPLICATION_HOST, FRONT_APPLICATION_PORT),
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

	Router.Group("/customer/")
	{
		Router.POST("/login/", rest.LoginRestController)
		Router.POST("/logout/", rest.LogoutRestController)

		Router.POST("/create/", rest.CreateCustomerRestController)
		Router.PUT("/update/", rest.UpdateCustomerRestController)
		Router.DELETE("/delete/", rest.DeleteCustomerRestController)
	}

	// Virtual Machines Rest API Endpoints
	Router.Group("/vm/")
	{
		Router.Use(middlewares.JwtAuthenticationMiddleware())
		{
			Router.POST("/deploy/", rest.DeployNewVirtualMachineRestController)
			Router.PUT("/update/config/", rest.UpdateVirtualMachineConfigurationRestController)
			Router.DELETE("/shutdown/", rest.ShutdownVirtualMachineRestController)
			Router.DELETE("/remove/", rest.RemoveVirtualMachineRestController)
		}

		Router.Use(middlewares.IsVirtualMachineOwnerMiddleware())
		{
			Router.GET("/get/list/", rest.GetCustomerVirtualMachinesRestController)
			Router.GET("/get/", rest.GetCustomerVirtualMachineInfoRestController)
		}
	}

	Router.Group("/resources/")
	{
		Router.Use(middlewares.JwtAuthenticationMiddleware())
		{
			Router.POST("/get/suggestions/", rest.GetAvailableResourcesInfoRestController)
		}
	}

	// Support Rest API Endpoints

	Router.Group("/support/")
	{
		Router.Use(middlewares.JwtAuthenticationMiddleware())
		{
			Router.POST("/feedback/", rest.SupportRestController)
		}
	}

	NotifyContext, CancelFunc := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)

	defer CancelFunc()
	Exception := httpServer.ListenAndServe()

	if errors.Is(Exception, http.ErrServerClosed) {
		NotifyContext.Done()
	} else {
		NotifyContext.Done()
	}
}

func (this *Server) Shutdown(Context context.Context, ServerInstance http.Server) {
	select {
	case <-Context.Done():
		ServerInstance.Shutdown(context.Background())
	}
}

func main() {
	DebugLogger.Printf("Running Http Application Server...")
	httpServer := NewServer(APPLICATION_HOST, APPLICATION_PORT)
	httpServer.Run()
}
