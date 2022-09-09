package loadbalancer

import (
	"fmt"
	"net/http"
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
	file, _ := os.OpenFile("Main.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
}

type LoadBalancerService struct {
	ServiceName       string
	ServiceEndpointIP string
}

func NewLoadBalancerService() *LoadBalancerService {
	return &LoadBalancerService{}
}

type LoadBalancerRoute struct {
	RouteUrl    string
	ServiceName string
}

func NewRoute() *LoadBalancerRoute {
	return &LoadBalancerRoute{}
}

type RouteParams struct {
	// Class, that represents info about the proxy route to the Virtual Machine
	// Server, allows to configure policies, to make a proxy server have a specific behaviour for that
	LoadBalancerServiceId string            `json:"LoadBalancerServiceId"`
	Headers               map[string]string `json:"Headers"`
	UpstreamConfig        struct {
		UpstreamName       string `json:"UpstreamName"`
		VirtualMachineHost string `json:"VirtualMachineHost"` // Host of the Virtual Machine to Proxy Traffic to
	} `json:"UpstreamConfig"`
}

func NewRouteParams(Headers map[string]string, UpstreamConfig struct {
	UpstreamName       string `json:"UpstreamName"`
	VirtualMachineHost string `json:"VirtualMachineHost"`
}) *RouteParams {
	return &RouteParams{
		Headers:        Headers,
		UpstreamConfig: UpstreamConfig,
	}
}

type LoadBalancerInfoServiceManager struct {
	// Service, that provides info about the given load balancer service
	LoadBalancerHost string `json:"LoadBalancerHost"`
	LoadBalancerPort string `json:"LoadBalancerPort"`
}

func NewLoadBalancerInfoServiceManager() *LoadBalancerInfoServiceManager {
	return &LoadBalancerInfoServiceManager{}
}

func (this *LoadBalancerInfoServiceManager) GetService(ServiceName string) (*LoadBalancerService, error) {
	// Returns the Array of the Avaiablle Services for the Load Balancer
}

func (this *LoadBalancerInfoServiceManager) GetRoutes(ServiceName string) ([]LoadBalancerRoute, error) {
	// Returns List of the Routes
}

type LoadBalancerServiceManager struct {
	// Manages the Services of the Specific Load Balancer
	HostMachineIPAddress string
}

func NewLoadBalancerServiceManager() *LoadBalancerServiceManager {
	return &LoadBalancerServiceManager{}
}

func (this *LoadBalancerServiceManager) ExposeNewService() LoadBalancerService {
	// Exposes new Load Balancer Traefik Service
}
func (this *LoadBalancerServiceManager) DestroyService() (bool, error) {
	// Destroy the Load Balancer Traefik Service
}

type LoadBalancerRouteManager struct {
	// Manages the Routes of the Specific Service
	HostMachineIPAddress string
}

func NewLoadBalancerRouteManager() *LoadBalancerRouteManager {
	return &LoadBalancerRouteManager{}
}

func (this *LoadBalancerRouteManager) AddNewRoute(RouteParams RouteParams) (LoadBalancerRoute, error) {
	// Adds new Route to the Traefik Load Balancer Service
}
func (this *LoadBalancerRouteManager) RemoveRoute(ServiceName string, RouteName string) (bool, error) {
	// Removes the Existing Route from the Traefik Load Balancer Service
}

type InternalLoadBalancerManager struct {
	Client client.Client
}

func NewInternalLoadBalancer(Host string, Port string) *InternalLoadBalancerManager {
	return &InternalLoadBalancerManager{}
}

func (this *InternalLoadBalancerManager) GetHostConnection(HostMachineIPAddress string) (client.Client, error) {
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

func (this *InternalLoadBalancerManager) AddNewDomainRoute(HostMachineIP string, RouteParams RouteParams) (bool, error) {
	// Adds New Virtual Machine Domain Route to the Global Web server to make it available
	// Goes to the Host Machine, where the Customer Deployed their application
	// Finding the Global Host Webserver, that serves all of the virtual machines across this Host Machine
	// and Simply Adds New Route

	// Receiving Traefik Connection
}
func (this *InternalLoadBalancerManager) RemoveDomainRoute(RouteParams RouteParams) {
	// Parsing Configuration of the Existing Load Balancer
}
