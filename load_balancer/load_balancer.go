package load_balancer

import (
	"encoding/json"
	"fmt"
	"io"

	"io/ioutil"
	"net/http"
	"net/url"

	"os"

	"github.com/docker/docker/client"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	DATACENTER_DOCKER_VERSION = os.Getenv("DATACENTER_DOCKER_VERSION") // Version of the Docker on the Datacenter Host Machine
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("LoadBalancerLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
}

type LoadBalancer struct {
	LoadBalancerHost string `json:"LoadBalancerHost" xml:"LoadBalancerHost"` // Host of where the Load Balancer is Running (usually the host of the host machine)
	LoadBalancerPort string `json:"LoadBalancerPort" xml:"LoadBalancerPort"` // Port of the Load Balancer
}

func NewLoadBalancer(Host string, Port string) *LoadBalancer {
	return &LoadBalancer{
		LoadBalancerHost: Host,
		LoadBalancerPort: Port,
	}
}

type RouteParams struct {
	// Class, that represents info about the proxy route to the Virtual Machine
	// Server, allows to configure policies, to make a proxy server have a specific behaviour for that
	Headers map[string]string `json:"Headers"`
	// Configuration of where the Load Balancer going to proxy traffic to
	UpstreamConfig struct {
		VirtualMachinePort string `json:"VirtualMachinePort" xml:"VirtualMachinePort"`
		VirtualMachineHost string `json:"VirtualMachineHost" xml:"VirtualMachineHost"` // Host of the Virtual Machine to Proxy Traffic to
	} `json:"LoadBalancerConfiguration" xml:"LoadBalancerConfiguration"`
}

func NewRouteParams(UpstreamConfig struct {
	VirtualMachinePort string `json:"VirtualMachinePort" xml:"VirtualMachinePort"`
	VirtualMachineHost string `json:"VirtualMachineHost" xml:"VirtualMachineHost"`
}, Headers ...map[string]string) *RouteParams {
	return &RouteParams{
		Headers:        Headers[0],
		UpstreamConfig: UpstreamConfig,
	}
}

type LoadBalancerManager struct {
	Client client.Client
}

func NewLoadBalancerManager() *LoadBalancerManager {
	return &LoadBalancerManager{}
}

func (this *LoadBalancerManager) GetHostConnection(HostMachineIPAddress string) (client.Client, error) {
	// Returns The Connection to the Host Machine, where the Customer's Virtual Machine Is Running on
	httpClient := http.DefaultClient
	HttpHeaders := map[string]string{
		"Access-Control-Allow-Origin":      "*",
		"Access-Control-Allow-Credentials": "true",
	}
	DockerClient, DockerErr := client.NewClient(
		fmt.Sprintf("unix://%s/var/run/docker.sock", HostMachineIPAddress),
		DATACENTER_DOCKER_VERSION,
		httpClient, HttpHeaders)

	return *DockerClient, DockerErr
}

func (this *LoadBalancerManager) AddNewDomainRoute(HostMachineIP string, RouteParams RouteParams) (*LoadBalancer, error) {
	// Creates New Load Balancer Service on the Remote Host Machine, where Customer's Virtual Machine Server
	// Is Being run,
	// This method calls, once the new Virtual Machine Server is being created

	newClient := http.DefaultClient
	var APIUrl = url.URL{
		Path: "/create/web/server/",
		Host: HostMachineIP,
	}
	newHttpRequest, _ := http.NewRequest("POST", APIUrl.String(), io.MultiReader())
	newHttpRequest.PostForm.Set("ProxyHost", RouteParams.UpstreamConfig.VirtualMachineHost)
	newHttpRequest.PostForm.Set("ProxyPort", RouteParams.UpstreamConfig.VirtualMachinePort)
	Response, ResponseError := newClient.Do(newHttpRequest)

	if ResponseError != nil {
		Logger.Error(
			"Load Balancer Initialization Service Responded With Error",
			zap.Error(ResponseError))
		return nil, ResponseError
	}

	var NewLoadBalancer LoadBalancer
	decodedResponseData, _ := ioutil.ReadAll(Response.Body)
	DeserializationError := json.Unmarshal(decodedResponseData, &NewLoadBalancer)
	return &NewLoadBalancer, DeserializationError

}
func (this *LoadBalancerManager) RemoveDomainRoute(HostMachineIP string, RouteParams RouteParams) (bool, error) {
	// Removes the Existing Load Balancer Server, that serves Virtual Machine Server
	// This method usually being called, once the Virtual Machine is being deleted or removed by Customer
	newClient := http.DefaultClient
	var APIUrl = url.URL{
		Path: "/remove/web/server/",
		Host: HostMachineIP,
	}
	newHttpRequest, _ := http.NewRequest("POST", APIUrl.String(), io.MultiReader())
	newHttpRequest.PostForm.Set("ProxyHost", RouteParams.UpstreamConfig.VirtualMachineHost)
	newHttpRequest.PostForm.Set("ProxyPort", RouteParams.UpstreamConfig.VirtualMachinePort)
	Response, ResponseError := newClient.Do(newHttpRequest)

	if ResponseError != nil || Response.StatusCode != 201 {
		Logger.Error(
			"Load Balancer Initialization Service Responded With Error",
			zap.Error(ResponseError))
		return false, ResponseError
	}
	return true, nil
}
