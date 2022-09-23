package host_search_rest

import (
	"net/http"
	"os"

	host_search "github.com/LovePelmeni/Infrastructure/host_search"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {
	// Initializes Logger
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("HostMachineSearchRestLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
}

func FindHostMachineRestController(Context *gin.Context) {
	// Rest Controller, that finds appropriate machine, based on the Resource Requirements
	ResourceRequirements := host_search.NewHostMachineRequirements()
	HostSearchManager := host_search.NewHostMachineSearcher()
	AvailableHostMachines, ParseError := HostSearchManager.GetAllHostMachines()

	if ParseError != nil {
		Logger.Error("Failed to get list of Available Host Machines", zap.Error(ParseError))
		Context.JSON(http.StatusOK, gin.H{"Error": "No Available Host Machines"})
		return
	}
	// Finding the Host Machine, based on the Customer Requirements
	HostMachine := HostSearchManager.SearchHostMachine(AvailableHostMachines, ResourceRequirements)
	Context.JSON(http.StatusOK, gin.H{
		"HostMachineClient":   HostMachine.Client.SourceIpAddress,
		"HostMachineUser":     HostMachine.Client.SourceUser,
		"HostMachinePassword": HostMachine.Client.SourcePassword,
	})
}
