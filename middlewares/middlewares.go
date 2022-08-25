package middlewares

import (
	"fmt"
	"net/http"

	"os"
	"strconv"

	"strings"
	"time"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/models"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var RedisClient *redis.Client

func init() {
	DatabaseNumber, _ := strconv.Atoi(os.Getenv("CACHE_STORAGE_DATABASE_NUMBER"))
	Client := redis.NewClient(&redis.Options{
		Password: os.Getenv("CACHE_STORAGE_DATABASE_PASSWORD"),
		Addr: fmt.Sprintf("%s:%s", os.Getenv("CACHE_STORAGE_DATABASE_HOST"),
			os.Getenv("CACHE_STORAGE_DATABASE_PORT")),
		DB: DatabaseNumber,
	})
	RedisClient = Client
}

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

func RequestIdempotencyMiddleware() gin.HandlerFunc {
	// Middleware checks for the Idempotency Of the HTTP Requests
	// To Avoid Unpleasent Situations

	return func(RequestContext *gin.Context) {

		RequestNumber := RequestContext.GetHeader("X-Idempotency-Key")
		var UrlCacheKey string
		for _, Property := range RequestContext.Params {
			UrlCacheKey += fmt.Sprintf("&%s", Property)
		}
		if Result, Error := RedisClient.Get(UrlCacheKey + "=" + RequestNumber).Result(); Error != nil {
			RedisClient.Set(UrlCacheKey+"="+RequestNumber, "Request is Being Processed", 10*time.Minute)
		} else {
			MapValuesResponse := func() map[string]string {
				var Map map[string]string
				for Index, Key := range strings.Split(Result, "=")[0] {
					Map[string(Key)] = string(strings.Split(Result, "=")[1][Index])
				}
				return Map
			}()
			RequestContext.AbortWithStatusJSON(http.StatusNotModified, MapValuesResponse)
		}
	}
}

func AuthorizationRequiredMiddleware() gin.HandlerFunc {
	// Middleware checks for customer is being Authorized
	return func(context *gin.Context) {
		if len(context.GetHeader("jwt-token")) == 0 {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Authorized"})
		}

		if _, Error := authentication.GetCustomerJwtCredentials(
			context.GetHeader("jwt-token")); Error != nil {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Authorized"})
		}
	}
}

func NonAuthorizationRequiredMiddleware() gin.HandlerFunc {
	// Middleware checks for the Customer is not being authorized
	return func(context *gin.Context) {
		if len(context.GetHeader("jwt-token")) != 0 {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Authorized"})
		}

		if _, Error := authentication.GetCustomerJwtCredentials(
			context.GetHeader("jwt-token")); Error == nil {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Authorized"})
		}
	}
}
