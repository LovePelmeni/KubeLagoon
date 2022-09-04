package loadbalancer

import (
	"context"
	"errors"

	"fmt"
	"log"

	"net/http"
	"net/http/httputil"

	"net/url"
	"os"
	"time"
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

// Abstractions

type LoadBalancerParams interface {
	// Interface describes Initial Configuration Parameters for the Load Balancer
	// That is required, in order to Initialize It
}

type BaseLoadBalancer interface {
	// Base Abstraction, a Load Balancer
	Create(InitParams LoadBalancerParams) (LoadBalancerInfo, error)
	Recreate(InitParams LoadBalancerParams) (LoadBalancerInfo, error)
	Delete(LoadBalancerIPAddress string) (bool, error)
}

// Parameter Entities

type InternalLoadBalancerParams struct {
	LoadBalancerParams
	VirtualMachineIPAddress  string `json:"VirtualMachineIPAddress"`
	ExternalLoadBalancerHost string `json:"ExternalLoadBalancerHost"`
	ExternalLoadBalancerPort string `json:"ExternalLoadBalancerPort"`
}

func NewInternalLoadBalancerParams(VirtualMachineIPAddress string, ExternalLoadBalancerHost string, ExternalLoadBalancerPort string) *InternalLoadBalancerParams {
	return &InternalLoadBalancerParams{
		ExternalLoadBalancerHost: ExternalLoadBalancerHost,
		ExternalLoadBalancerPort: ExternalLoadBalancerPort,
	}
}

type ExternalLoadBalancerParams struct {
	// Represents Structure with Initial Parameters to Initialize Load Balancer
	LoadBalancerParams
	Host                     string `json:"Host"`                     // Host IP of the External Load Balancer
	Port                     string `json:"Port"`                     // Port of the External Load Balancer
	HostMachineHost          string `json:"HostMachineIPAddress"`     // Host of the Machine, where the Internal Load Balancer Is Running On
	InternalLoadBalancerPort string `json:"InternalLoadBalancerPort"` // Port of the Internal Load Balancer
}

func NewExternalLoadBalancerParams(VirtualMachineIPAddress string, HostMachineIPAddress string) *ExternalLoadBalancerParams {
	return &ExternalLoadBalancerParams{
		HostMachineHost:          VirtualMachineIPAddress,
		InternalLoadBalancerPort: HostMachineIPAddress,
	}
}

type LoadBalancerInfo struct {
	HealthInfo *LoadBalancerHealthInfo `json:"HealthInfo,omitempty;" xml:"HealthInfo"`
	Host       string                  `json:"Host" xml:"Host"`
	Port       string                  `json:"Port" xml:"Port"`
}

func NewLoadBalancerInfo(HealthInfo LoadBalancerHealthInfo, Host string, Port string) *LoadBalancerInfo {
	return &LoadBalancerInfo{
		HealthInfo: &HealthInfo,
		Host:       Host,
		Port:       Port,
	}
}

type LoadBalancerHealthInfo struct {
	IsAlive bool `json:"IPAddress"`
}

func NewLoadBalancerHealthInfo(IsAlive bool) *LoadBalancerHealthInfo {
	return &LoadBalancerHealthInfo{
		IsAlive: IsAlive,
	}
}

// Server Entities

type ExternalLoadBalancer struct {
	BaseLoadBalancer
}

func NewLoadBalancer() *ExternalLoadBalancer {
	return &ExternalLoadBalancer{}
}

func (this *ExternalLoadBalancer) Create(LoadBalancerInitParams ExternalLoadBalancerParams) (*LoadBalancerInfo, error) {
	// Initializes New Load Balancer

	InternalLoadBalancerUrl := url.URL{
		Path: "/",
		Host: LoadBalancerInitParams.HostMachineHost + fmt.Sprintf(
			":%s", LoadBalancerInitParams.InternalLoadBalancerPort), // Pointing to the Internal Load Balancer, That Is Running on the Host Machine
	}
	newLoadBalancer := httputil.NewSingleHostReverseProxy(&InternalLoadBalancerUrl)

	// Load Balancer Deployment

	LoadBalancerHTTPEndpoint := func(Response http.ResponseWriter, Request *http.Request) {
		DebugLogger.Printf("Serving new Internal Load Balancer")
		newLoadBalancer.ServeHTTP(Response, Request)
	}
	// Load Balancer Error Handler Http Endpoint
	LoadBalancerErrorHandler := func(ErrorResponse http.ResponseWriter, ErrorRequest *http.Request, Error error) {
		if Error != nil {
		}
	}
	newLoadBalancer.ErrorHandler = LoadBalancerErrorHandler

	ExternalLoadBalancerInfo := LoadBalancerInfo{
		Host: newLoadBalancer.Host,
		Port: newLoadBalancer.Port,
	}
	return &ExternalLoadBalancerInfo, nil
}

func (this *ExternalLoadBalancer) Recreate(LoadBalancerParams ExternalLoadBalancerParams) (*LoadBalancerInfo, error) {
	// Recreates Existing Load Balancer, if one goes down
	NewLoadBalancer, CreationError := this.Create(LoadBalancerParams)
	if CreationError != nil {
		ErrorLogger.Printf("Failed to Recreate External Load Balancer, Error: - %s", CreationError)
		return nil, CreationError
	}
	return NewLoadBalancer, nil
}

func (this *ExternalLoadBalancer) Delete(LoadBalancerIPAddress string) (bool, error) {
	// Deletes Existing Load Balancer
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*15)
	defer CancelFunc()

	Server := http.Server{
		Addr: LoadBalancerIPAddress,
	}
	Server.Shutdown(TimeoutContext)
	select {
	case <-time.After(15):
		return true, nil
	case <-TimeoutContext.Done():
		return false, errors.New("Failed to Remove External Load Balancer, Timeout Error")
	}
}

func (this *ExternalLoadBalancer) GetHealthInfo(LoadBalancerIPAddress string) (*LoadBalancerHealthInfo, error) {
	// Returns Health Info about the Load Balancer
	return &LoadBalancerHealthInfo{}, nil
}

type InternalLoadBalancer struct {
	BaseLoadBalancer
	Host string `json:"Host"` // localhost
	Port string `json:"Port"` // port of the internal load balancer server
}

func NewInternalLoadBalancer(Host string, Port string) *InternalLoadBalancer {
	return &InternalLoadBalancer{
		Host: Host,
		Port: Port,
	}
}

func (this *InternalLoadBalancer) Create(LoadBalancerInitParams InternalLoadBalancerParams) (*LoadBalancerInfo, error) {
	// Initializes New Load Balancer

	InternalLoadBalancerUrl := url.URL{
		Path: "/",
		Host: LoadBalancerInitParams.VirtualMachineIPAddress, // Pointing to the Virtual Machine, That Is Running on the Host Machine
	}
	newLoadBalancer := httputil.NewSingleHostReverseProxy(&InternalLoadBalancerUrl)

	// Load Balancer Deployment

	LoadBalancerHTTPEndpoint := func(Response http.ResponseWriter, Request *http.Request) {
		DebugLogger.Printf("Serving new Internal Load Balancer")
		newLoadBalancer.ServeHTTP(Response, Request)
	}
	// Load Balancer Error Handler Http Endpoint
	LoadBalancerErrorHandler := func(ErrorResponse http.ResponseWriter, ErrorRequest *http.Request, Error error) {
		if Error != nil {
		}
	}
	newLoadBalancer.ErrorHandler = LoadBalancerErrorHandler

	InternalLoadBalancerInfo := LoadBalancerInfo{
		Host: newLoadBalancer.Host,
		Port: newLoadBalancer.Port,
	}
	return &InternalLoadBalancerInfo, nil
}

func (this *InternalLoadBalancer) Recreate(LoadBalancerParams InternalLoadBalancerParams) (*LoadBalancerInfo, error) {
	// Recreates Existing Load Balancer, if one goes down
	NewLoadBalancer, CreationError := this.Create(LoadBalancerParams)
	if CreationError != nil {
		ErrorLogger.Printf("Failed to Recreate Internal Load Balancer, Error: - %s", CreationError)
		return nil, CreationError
	}
	return NewLoadBalancer, nil
}

func (this *InternalLoadBalancer) Delete(LoadBalancerIPAddress string) (bool, error) {
	// Deletes Existing Load Balancer
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*15)
	defer CancelFunc()

	Server := http.Server{
		Addr: LoadBalancerIPAddress,
	}
	Server.Shutdown(TimeoutContext)
	select {
	case <-time.After(15):
		return true, nil
	case <-TimeoutContext.Done():
		return false, errors.New("Failed to Remove Internal Load Balancer, Timeout Error")
	}
}

func (this *InternalLoadBalancer) GetHealthInfo(LoadBalancerIPAddress string) (*LoadBalancerHealthInfo, error) {
	// Returns Health Info about the Load Balancer
	return &LoadBalancerHealthInfo{}, nil
}
