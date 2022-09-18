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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var (
	Logger *zap.Logger
)

var (
	RedisClient *redis.Client
)

func InitializeProductionLogger() {
	// Initializing Zap Logger
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("Main.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
	DatabaseNumber, _ := strconv.Atoi(os.Getenv("CACHE_STORAGE_DATABASE_NUMBER"))
	Client := redis.NewClient(&redis.Options{
		Password: os.Getenv("CACHE_STORAGE_DATABASE_PASSWORD"),
		Addr: fmt.Sprintf("%s:%s", os.Getenv("CACHE_STORAGE_DATABASE_HOST"),
			os.Getenv("CACHE_STORAGE_DATABASE_PORT")),
		DB: DatabaseNumber,
	})
	RedisClient = Client
}

// VIRTUAL MACHINE MIDDLEWARES

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

func SetReadyOperationMiddleware() gin.HandlerFunc {
	// Sets status `Ready` to the Virtual Machine
	// being called only on HTTP Response
	return func(Context *gin.Context) {

		VirtualMachineId := Context.Query("VirtualMachineId")
		var VirtualMachine models.VirtualMachine

		models.Database.Model(&models.VirtualMachine{}).Where(
			"id = ?", VirtualMachineId).Find(&VirtualMachine)

		VirtualMachine.State = models.StatusNotReady
		VirtualMachine.Save()
		Context.Next()
	}
}

func SetNotReadyOperationMiddleware() gin.HandlerFunc {
	// Sets status `NotReady` to the Virtual Machine
	// being called only on HTTP Request
	return func(Context *gin.Context) {

		VirtualMachineId := Context.Query("VirtualMachineId")
		var VirtualMachine models.VirtualMachine

		models.Database.Model(&models.VirtualMachine{}).Where(
			"id = ?", VirtualMachineId).Find(&VirtualMachine)

		if Context.Request.Response.StatusCode != 0 || len(Context.Request.Response.Status) != 0 {
			VirtualMachine.State = models.StatusReady
			VirtualMachine.Save()
		}
		Context.Next()
	}
}

func IsReadyToPerformOperationMiddleware() gin.HandlerFunc {
	// Middlewares is used to prevent following case scenario

	// Suppose: Virtual Machine is Currently busy, and is applying new Configuration
	// So we need to add some blocker in order to prevent any critical operations on that specific
	// machine, to prevent corruption,
	// Virtual Machine Model has 2 states: `Ready` and `NotReady`

	// When `NotReady` State occurs, it means, that this Virtual Machine is already being used and performs another operation
	// So this Middleware is being used for detecting that State, before performing new Request to that Specific VM
	return func(Context *gin.Context) {

		var VirtualMachineId = Context.Query("VirtualMachineId")
		var VirtualMachine models.VirtualMachine

		models.Database.Model(&models.VirtualMachine{}).Where(
			"id = ?", VirtualMachineId).Find(&VirtualMachine)

		switch {

		case VirtualMachine.State == "NotReady":
			Context.AbortWithStatusJSON(http.StatusServiceUnavailable,
				gin.H{"Error": "this Server is already Performing other Operation, please Wait"})

		case VirtualMachine.State == "Ready":
			Context.Next()

		default:
			Context.Next()
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

// AUTHORIZATION MIDDLEWARES

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

// ---------------------------------------------

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
		// Checking if the Infrastructure is able to perform the Http Request, related to the VM
		TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
		defer CancelFunc()

		_, ConnectionError := govmomi.NewClient(TimeoutContext, APIUrl, false)
		if ConnectionError != nil {
			RequestContext.AbortWithStatusJSON(http.StatusServiceUnavailable,
				gin.H{"Error": "Service Is Currently Not Available, Try a bit Later"})
			return
		}
		RequestContext.Next()
	}
}

