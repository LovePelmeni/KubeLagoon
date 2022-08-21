package middlewares

import (
	"github.com/gin-gonic/gin"
)

func JwtAuthenticationMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		
	}
}

func IsVirtualMachineOwnerMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {

	}
}

func AuthorizationRequiredMiddleware() gin.HandlerFunc {
	return func(RequestContext *gin.Context) {
		
	}
}

func NonAuthorizationRequiredMiddleware() gin.HandlerFunc {
	return func(RequestContext *gin.Context) {

	}
}
