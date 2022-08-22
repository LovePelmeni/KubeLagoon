package healthcheck_rest

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/deploy"
	"github.com/LovePelmeni/Infrastructure/healthcheck"
	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
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
	Client govmomi.Client
)

// package provides Rest API Controllers, that Provides Info about the Virtual Machine Server Health Metrics

func GetVirtualMachineHealthMetricRestController(RequestContext *gin.Context) {

	type HealthMetric struct {
		Storage healthcheck.StorageInfo     `json:"StorageInfo"`
		State   healthcheck.AliveInfo       `json:"StateInfo"`
		Memory  healthcheck.MemoryUsageInfo `json:"MemoryInfo"`
		Cpu     healthcheck.CPUInfo         `json:"CpuInfo"`
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

	HealthCheckManager := healthcheck.NewVirtualMachineHealthCheckManager()
	HealthCheckMetrics := HealthMetric{
		Storage: HealthCheckManager.GetStorageUsageMetrics(),
		Cpu:     HealthCheckManager.GetCpuMetrics(),
		Memory:  HealthCheckManager.GetMemoryUsageMetrics(),
		State:   HealthCheckManager.GetAliveMetrics(),
	}
	RequestContext.JSON(http.StatusOK, gin.H{"Metrics": HealthCheckMetrics})
}
