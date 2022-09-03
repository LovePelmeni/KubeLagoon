package load_balanc_rest

import (
	"github.com/gin-gonic/gin"
)

func RecreateLoadBalancerRestController(Request *gin.Context) {
	// Rest Controller recreates Load Balancer Instance, without Touching existed Database Model
}

func CreateLoadBalancerRestController(Request *gin.Context) {
	// Rest Controller, that Creates New Load Balancer Instance, including the Model
}

func DeleteLoadBalancerRestController(Request *gin.Context) {
	// Rest Controller, that Deletes Load Balancer Instance, including the Model
}
