package loadbalancer

import (
	"context"
	"fmt"
	"strings"

	"log"

	"net/http"

	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

var (
	DATACENTER_DOCKER_VERSION = os.Getenv("DATACENTER_DOCKER_VERSION") // Version of the Docker on the Datacenter Host Machine
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("LoadBalancer.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if Error != nil {
		panic(Error)
	}
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Package Consists of the API for managing the Load Balancers / Reverse Proxy, that handles
// HTTP Traffic to the Virtual Machine Servers from outside

type InternalLoadBalancerConfiguration struct {

	// General Info

	LoadBalancerName     string `json:"LoadBalancerName" xml:"LoadBalancerName"`         // Load Balancer Name - "NGINX" or "APACHE" ONLY 2 Options Available Now
	HostMachineIPAddress string `json:"HostMachineIPAddress" xml:"HostMachineIPAddress"` // on what machine to deploy Load Balancer

	// Info about the Load Balancer Location

	InternalLoadBalancerHost string `json:"ExternalLoadBalancerHost"` // Host of the Internal Load Balancer (`localhost`)
	InternalLoadBalancerPort string `json:"ExternalLoadBalancerPort"` // Port of the Internal Load Balancer (`picked up by the Port Manager`)

	// Info about where the Load balancer going to Proxy the Requests

	ProxyHost string `json:"ProxyHost"` // Virtual Machine IP Address Internal Load Balancer, going to proxy request to
	ProxyPort string `json:"ProxyPort"` // Port where the Load Balancer Going to Proxy Requests to
}

func NewInternalLoadBalancerParams(VirtualMachineIPAddress string, ExternalLoadBalancerHost string, ExternalLoadBalancerPort string) *InternalLoadBalancerParams {
	return &InternalLoadBalancerConfiguration{
		InternalLoadBalancerHost: ExternalLoadBalancerHost,
		InternalLoadBalancerPort: ExternalLoadBalancerPort,
	}
}

type LoadBalancerConfigurationSelector struct {
	// Server Returns Configuration + Dockerfile for the Web Server, Selected By the Customer
}

func NewLoadBalancerSelector() *LoadBalancerConfigurationSelector {
	return &LoadBalancerConfigurationSelector{}
}

func (this *LoadBalancerConfigurationSelector) GetLoadBalancerFile(Configuration InternalLoadBalancerConfiguration) string {
	// Returns Configuration File of the Load Balancer, based on the Configuraion, passed by the Customer
	if strings.ToLower(Configuration.LoadBalancerName) == "nginx" {
		// Returning the NGINX Configuration File
		return this.GetNginxLoadBalancerDockerFile(Configuration)
	}
	if strings.ToLower(Configuration.LoadBalancerName) == "apache" {
		// Returning the APACHE Configuraion File
		return this.GetApacheLoadBalancerDockerFile(Configuration)
	}
}
func (this *LoadBalancerConfigurationSelector) GetNginxLoadBalancerDockerFile(LoadBalancerConfiguration InternalLoadBalancerConfiguration) string {
	//
	newDockerFile := client.DefaultDockerHost
	NginxLoadBalancerConfiguration := fmt.Sprintf(`
	events {
		worker_connections 1024; 
	}
	http {
		upstream application_upstream {
			server %s:%s; // the address server application http server is running on; 
		}
		server {
			listen 80;
			location / {
				proxy_pass http://application_upstream; 
				proxy_http_version                 1.1; 
				proxy_set_header Host       $http_host;
				proxy_set_header Upgrade $http_upgrade;
				add_header "Access-Control-Allow-Origin" $http_origin; 
				add_header "Access-Control-Allow-Methods" "GET,POST,DELETE,PUT,OPTIONS";
				add_header "Access-Control-Allow-Credentials" "true";
			}
		}
	}
`)
}
func (this *LoadBalancerConfigurationSelector) GetApacheLoadBalancerDockerFile(Configuration InternalLoadBalancerConfiguration) string {
	// Returns the Dockerfile for the APACHE Load Balancer, with the Configuration, passed by the Customer

}

// Server Entities

type InternalLoadBalancerManager struct{}

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

func (this *InternalLoadBalancerManager) CreateLoadBalancer(Configuration InternalLoadBalancerConfiguration) (*LoadBalancerInfo, error) {

	// * Creates Load Balancer Server on the Remote Host Machine using Docker
	// * Requires Dockerfile with the WebServer, that is already configured

	// Initializing Docker Client

	DockerClient, Err := this.GetHostConnection(Configuration.HostMachineIPAddress)
	if Err != nil {
		return nil, Err
	}

	// Getting Load Balancer Dockerfile, based on the Configuration, passed by the Customer

	LbSelector := NewLoadBalancerSelector()
	WebServerDockerFile := LbSelector.GetLoadBalancerFile(Configuration)

	LbTimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*30)
	defer CancelFunc()

	LoadBalancerContainer, CreationError := DockerClient.ContainerCreate(
		LbTimeoutContext,
		&container.Config{},
		&container.HostConfig{AutoRemove: true},
		&network.NetworkingConfig{},
	)

	return &LoadBalancerInfo, ResponseBodyParseError
}

func (this *InternalLoadBalancerManager) RecreateLoadBalancer(Configuration InternalLoadBalancerConfiguration) (*LoadBalancerInfo, error) {
	// Recreates Existing Load Balancer, if one goes down
	// Uses the `Create` Method to perform the Operation of the Creating new Load Balancer

	NewLoadBalancer, CreationError := this.CreateLoadBalancer(LoadBalancerParams)
	if CreationError != nil {
		ErrorLogger.Printf("Failed to Recreate Internal Load Balancer, Error: - %s", CreationError)
		return nil, CreationError
	}
	return NewLoadBalancer, nil
}

func (this *InternalLoadBalancerManager) GetHealthInfo(LoadBalancerIPAddress string) (*LoadBalancerHealthInfo, error) {
	// Returns Health Info about the Load Balancer
	return &LoadBalancerHealthInfo{}, nil
}
