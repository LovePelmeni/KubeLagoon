package healthcheck_rest

import (
	"context"

	"net/http"
	"net/url"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/healthcheck"
	
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"

	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25/mo"
)

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

var (
	Client *govmomi.Client
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("HealthCheckLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	// Initializing Govmomi Client for the VM Server

	var RestClient *rest.Client
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	APIClient, ConnectionError := govmomi.NewClient(TimeoutContext, APIUrl, false)
	switch {
	case ConnectionError != nil:
		Logger.Error("FAILED TO INITIALIZE CLIENT, DOES THE VMWARE HYPERVISOR ACTUALLY RUNNING?")

	case ConnectionError == nil:
		RestClient = rest.NewClient(APIClient.Client)
		if FailedToLogin := RestClient.Login(TimeoutContext, APIUrl.User); FailedToLogin != nil {
			Logger.Error("FAILED TO LOGIN TO THE VMWARE HYPERVISOR SERVER", zap.Error(FailedToLogin))
		}
	}
	Client = APIClient
}

// package consists of Rest API Controllers, that Provides Info about the Virtual Machine Server Health Metrics

func GetVirtualMachineHealthMetricRestController(RequestContext *gin.Context) {

	type HealthMetric struct {
		Storage    healthcheck.StorageInfo     `json:"StorageInfo"`
		State      healthcheck.AliveInfo       `json:"StateInfo"`
		Memory     healthcheck.MemoryUsageInfo `json:"MemoryInfo"`
		Cpu        healthcheck.CPUInfo         `json:"CpuInfo"`
		HostSystem healthcheck.HostSystemInfo  `json:"HostSystemInfo"`
	}

	// Receiving Virtual Machine Instance
	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	VmOwnerId := RequestContext.Query("CustomerId")

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	Collector := property.DefaultCollector(Client.Client)
	VirtualMachineManager := deploy.NewVirtualMachineManager(*Client.Client)

	var VirtualMachine mo.VirtualMachine

	VirtualMachineRef, FindError := VirtualMachineManager.GetVirtualMachine(VirtualMachineId, VmOwnerId)
	RetrieveError := Collector.RetrieveOne(TimeoutContext, VirtualMachineRef.Reference(), []string{"*"}, &VirtualMachine)

	if FindError != nil || RetrieveError != nil {
		RequestContext.JSON(http.StatusOK, gin.H{"Error": "Virtual Machine Not Found"})
	}

	HealthCheckManager := healthcheck.NewVirtualMachineHealthCheckManager(&VirtualMachine)
	HealthCheckMetrics := HealthMetric{
		Storage:    HealthCheckManager.GetStorageUsageMetrics(),
		Cpu:        HealthCheckManager.GetCpuMetrics(),
		Memory:     HealthCheckManager.GetMemoryUsageMetrics(),
		State:      HealthCheckManager.GetAliveMetrics(),
		HostSystem: HealthCheckManager.GetHostSystemHealthMetrics(),
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Metrics": HealthCheckMetrics})
}
