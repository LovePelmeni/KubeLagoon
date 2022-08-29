package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"os"
	"strconv"

	"strings"
	"time"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/vmware/govmomi"

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
		if len(context.GetHeader("Authorization")) == 0 {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Unauthorized"})
			return
		}

		if _, Error := authentication.GetCustomerJwtCredentials(
			context.GetHeader("Authorization")); Error != nil {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "Unauthorized"})
			return
		}
		context.Next()
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
		context.Next()
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
			RequestContext.Next()
		} else {
			MapValuesResponse := func() map[string]string {
				var Map map[string]string
				for Index, Key := range strings.Split(Result, "=")[0] {
					Map[string(Key)] = string(strings.Split(Result, "=")[1][Index])
				}
				return Map
			}()
			RequestContext.AbortWithStatusJSON(http.StatusNotModified, MapValuesResponse)
			return
		}
	}
}

func AuthorizationRequiredMiddleware() gin.HandlerFunc {
	// Middleware checks for customer is being Authorized
	return func(context *gin.Context) {
		if len(context.GetHeader("Authorization")) == 0 {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "You are Not Authorized"})
			return
		}

		if _, Error := authentication.GetCustomerJwtCredentials(
			context.GetHeader("Authorization")); Error != nil {
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "You are Not Authorized"})
			return
		}
		context.Next()
	}
}

func NonAuthorizationRequiredMiddleware() gin.HandlerFunc {
	// Middleware checks for the Customer is not being authorized
	return func(context *gin.Context) {
		if len(context.GetHeader("Authorization")) != 0 {
			fmt.Printf(context.GetHeader("Authorization"))
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "You are Authorized"})
			return
		}
		if _, Error := authentication.GetCustomerJwtCredentials(
			context.GetHeader("Authorization")); Error == nil && len(context.GetHeader("Authorization")) != 0 {
			fmt.Printf(context.GetHeader("Authorization"))
			context.AbortWithStatusJSON(
				http.StatusForbidden, gin.H{"Error": "You are Authorized"})
			return
		}
		context.Next()
	}
}

func InfrastructureHealthCircuitBreakerMiddleware() gin.HandlerFunc {
	// Middleware, that checks for VM Server Accessibility, when the Request Associated within It
	// is being Requested, If the Datacenter is currently not available, it will respond with
	// not available Exception
	return func(RequestContext *gin.Context) {

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
		TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
		defer CancelFunc()

		_, ConnectionError := govmomi.NewClient(TimeoutContext, APIUrl, false)
		if ConnectionError != nil {
			RequestContext.AbortWithStatusJSON(http.StatusServiceUnavailable,
				gin.H{"Error": "Service Is Currently Not Available, Try Later"})
			return
		}
		RequestContext.Next()
	}
}
