package load_balanc_rest

import (
	"github.com/gin-gonic/gin"
)

func ExposeProxyServiceRestController(Request *gin.Context) {
	// Exposes new Proxy Service on the Traefik Server on the remote host machine, that will be hanlding
	// Requests to the Specific Virtual Machine (s)
}
func DestroyProxyServiceRestController(Request *gin.Context) {
	// Destroys the Existing Proxy Service
}

func AddProxyRouteRestController(Request *gin.Context) {
	// Rest Controller, that adds new Route Record to the Host Load Balancer
	// So the Virtual Machine can be accessed
}

func RemoveProxyRouteRestController(Request *gin.Context) {
	// Rest Controller, that removes existing Route Record from the Load Balancer
	// So no one will be able to access it from outside
}
