package main

import (
	"context"
	"errors"

	"fmt"
	"net/http"

	"os"
	"os/signal"
	"syscall"

	"github.com/LovePelmeni/Infrastructure/rest"
	"github.com/gin-gonic/gin"
)

var (
	APPLICATION_HOST = os.Getenv("APPLICATION_HOST")
	APPLICATION_PORT = os.Getenv("APPLICATION_PORT")
)

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
		Router.POST("/deploy/", rest.DeployNewVirtualMachineRestController)
		Router.PUT("/update/config/", rest.UpdateVirtualMachineConfigurationRestController)
		Router.DELETE("/shutdown/", rest.ShutdownVirtualMachineRestController)
		Router.DELETE("/remove/", rest.RemoveVirtualMachineRestController)
	}

	Router.Group("/resources/")
	{
		Router.POST("/get/suggestions/")
	}

	// Support Rest API Endpoints

	Router.Group("/support/")
	{
		Router.POST("/feedback/", rest.SupportRestController)
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
	httpServer := NewServer(APPLICATION_HOST, APPLICATION_PORT)
	httpServer.Run()
}
