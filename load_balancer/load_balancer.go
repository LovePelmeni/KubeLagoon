package loadbalancer

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"log"

	"net/http"

	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
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

type LoadBalancerContainerBuilder struct {
	// Serves the Load Balancers Containers
	Client *client.Client
}

func NewLoadBalancerContainerBuilder(Client *client.Client) *LoadBalancerContainerBuilder {
	return &LoadBalancerContainerBuilder{
		Client: Client,
	}
}
func (this *LoadBalancerContainerBuilder) CreateContainer(ContainerName string, ImageName string, Configuration InternalLoadBalancerConfiguration) {
	// Creates New Container out of the Given Load Balancer Image
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()
	this.Client.ContainerCreate(TimeoutContext,
		&container.Config{
			WorkingDir:   "/",
			Image:        ImageName,
			Tty:          false,
			ExposedPorts: nat.PortSet{nat.Port(Configuration.InternalLoadBalancerPort): struct{}{}},
		},
		&container.HostConfig{
			AutoRemove: true,
		},
		&network.NetworkingConfig{}, &v1.Platform{}, ContainerName)

}
func (this *LoadBalancerContainerBuilder) DeleteContainer() {
	// Deletes the Existing Load Balancer Container
}
func (this *LoadBalancerContainerBuilder) RecreateContainer() {
	// Recreates New Load Balancer Container, based on the Image
}

type LoadBalancerImageBuilder struct {
	// Server Returns Configuration + Dockerfile for the Web Server, Selected By the Customer
	Client *client.Client
}

func NewLoadBalancerBuilder() *LoadBalancerImageBuilder {
	return &LoadBalancerImageBuilder{}
}

func (this *LoadBalancerImageBuilder) BuildLoadBalancerImage(Configuration InternalLoadBalancerConfiguration) (types.ImageBuildResponse, error) {
	// Returns Configuration File of the Load Balancer, based on the Configuraion, passed by the Customer
	if strings.ToLower(Configuration.LoadBalancerName) == "nginx" {
		// Returning the NGINX Configuration File
		return this.BuildNginxLoadBalancerImage(Configuration)
	}
	if strings.ToLower(Configuration.LoadBalancerName) == "apache" {
		// Returning the APACHE Configuraion File
		return this.BuildApacheLoadBalancerImage(Configuration)
	}
}

func (this *LoadBalancerImageBuilder) BuildNginxLoadBalancerImage(LoadBalancerConfiguration InternalLoadBalancerConfiguration) (types.ImageBuildResponse, error) {
	//	Returns the Custom NGINX Load Balancer Image, based on the Configuration, that customer has passed

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	Dockerfile := fmt.Sprintf(`
		FROM nginx 
		VOLUMES /etc/nginx/nginx.conf/ ./nginx.conf 
		EXPOSE %s
	`, LoadBalancerConfiguration.InternalLoadBalancerPort)

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
	LoadBalancerImageTag := fmt.Sprintf("nginx-load-balancer-%s-%s",
		LoadBalancerConfiguration.HostMachineIPAddress, LoadBalancerConfiguration.ProxyHost)

	WrittenContent := bufio.Writer{}
	WrittenContent.WriteString(NginxLoadBalancerConfiguration)

	NewDockerImage, BuildError := this.Client.ImageBuild(TimeoutContext,
		bufio.NewReader(WrittenContent), types.ImageBuildOptions{

			Tags:        []string{LoadBalancerImageTag},
			Dockerfile:  Dockerfile,
			Remove:      true,
			ForceRemove: true,
		})
	return NewDockerImage, BuildError
}

func (this *LoadBalancerImageBuilder) BuildApacheLoadBalancerImage(Configuration InternalLoadBalancerConfiguration) (types.ImageBuildResponse, error) {
	// Returns the Dockerfile Image for the APACHE Load Balancer, with the Configuration, passed by the Customer

	Dockerfile := fmt.Sprintf(`
		FROM ubuntu 
		RUN apt update 
		RUN apt install –y apache2 
		RUN apt install –y apache2-utils 
		RUN apt clean 
		EXPOSE %s
		CMD [“apache2ctl”, “-D”, “FOREGROUND”]
	`, Configuration.InternalLoadBalancerPort)

	ApacheConfigurationFile := fmt.Sprintf(`

	`)

	LoadBalancerTimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	Tag := fmt.Sprintf("apache-load-balancer-%s-%s",
		Configuration.HostMachineIPAddress, Configuration.ProxyHost)

	LoadBalancerImage, BuildError := this.Client.ImageBuild(
		LoadBalancerTimeoutContext, &bufio.Reader{}, types.ImageBuildOptions{
			Dockerfile:  Dockerfile,
			Tags:        []string{Tag},
			Remove:      true,
			ForceRemove: true,
		})
	return LoadBalancerImage, BuildError
}

// Server Entities

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

func (this *InternalLoadBalancerManager) CreateLoadBalancer(Configuration InternalLoadBalancerConfiguration) (*LoadBalancerInfo, error) {

	// * Creates Load Balancer Server on the Remote Host Machine using Docker
	// * Requires Dockerfile with the WebServer, that is already configured

	// Initializing Docker Client

	DockerClient, Err := this.GetHostConnection(Configuration.HostMachineIPAddress)
	if Err != nil {
		return nil, Err
	}

	// Getting Load Balancer Dockerfile, based on the Configuration, passed by the Customer

	LbImageBuilder := NewLoadBalancerImageBuilder(DockerClient)
	LbContainerBuilder := NewLoadBalancerContainerBuilder(&DockerClient)

	WebServerDockerFile, BuildError := LbImageBuilder.BuildLoadBalancerImage(Configuration)

	if BuildError != nil {
		return nil, BuildError
	}

	LoadBalancerContainer, CreationError := LbContainerBuilder.CreateContainer()

	if CreationError != nil {
		return nil, CreationError
	}

	LoadBalancerInfo := LoadBalancerInfo{
		Host: Configuration.InternalLoadBalancerHost,
		Port: Configuration.InternalLoadBalancerPort,

		ProxyHost: Configuration.ProxyHost,
		ProxyPort: Configuration.ProxyPort,
	}
	return &LoadBalancerInfo
}

func (this *InternalLoadBalancerManager) RecreateLoadBalancer(Configuration InternalLoadBalancerConfiguration) (*LoadBalancerInfo, error) {
	// Recreates Existing Load Balancer, if one goes down
	// Uses the `Create` Method to perform the Operation of the Creating new Load Balancer

	NewLoadBalancer, CreationError := this.CreateLoadBalancer(Configuration)
	if CreationError != nil {
		ErrorLogger.Printf("Failed to Recreate Internal Load Balancer, Error: - %s", CreationError)
		return nil, CreationError
	}
	return NewLoadBalancer, nil
}
