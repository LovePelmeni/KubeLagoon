package middlewares

import (
	"net/http"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/gin-gonic/gin"
)

func JwtAuthenticationMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		if len(context.GetHeader("jwt-token")) == 0 {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Unauthorized"})
		}

		if _, Error := authentication.GetCustomerJwtCredentials(
			context.GetHeader("jwt-token")); Error != nil {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Unauthorized"})
		}
	}
}

func IsVirtualMachineOwnerMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		VirtualMachineId := context.Query("VirtualMachineId")
		OwnerId := context.Query("OwnerId")
		if Exists := models.Database.Model(
			&models.VirtualMachine{}).Where("id = ? AND owner_id = ?", VirtualMachineId, OwnerId); Exists.Error != nil {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Status": "You're not the Owner of this VM"})
			return
		}
	}
}

func IdempotencyMiddleware() gin.HandlerFunc {
	return func(RequestContext *gin.Context){
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
