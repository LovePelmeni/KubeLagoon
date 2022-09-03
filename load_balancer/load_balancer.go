package loadbalancer

// Package Consists of the API for managing the Load Balancers / Reverse Proxy, that handles
// HTTP Traffic to the Virtual Machine Servers from outside

// Abstractions

type BaseLoadBalancer interface {
	// Base Abstraction, a Load Balancer
	Create(InitParams InitializationParams) (LoadBalancerInfo, error)
	GetInfo(LoadBalancerIPAddress string) (LoadBalancerInfo, error)
	Delete(LoadBalancerIPAddress string) (bool, error)
}

type BaseLoadBalancerManager interface {
	// Base Abstraction, that represents Manager for "Managing" Load Balancers
	CreateLoadBalancer(InitParams InitializationParams) (LoadBalancerInfo, error)
	DeleteLoadBalancer(LoadBalancerIPAddress string) (bool, error)
	GetInfo(LoadBalancerIPAddress string) (LoadBalancerInfo, error)
	GetHealthInfo(LoadBalancerIPAddress string) (LoadBalancerHealthInfo, error)
}

// Parameter Entities

type InitializationParams struct {
	// Represents Structure with Initial Parameters to Initialize Load Balancer
	VirtualMachineIPAddress string `json:"VirtualMachineIPAddress"`
	HostMachineIPAddress    string `json:"HostMachineIPAddress"`
}

func NewInitializationParams(VirtualMachineIPAddress string, HostMachineIPAddress string) *InitializationParams {
	return &InitializationParams{
		VirtualMachineIPAddress: VirtualMachineIPAddress,
		HostMachineIPAddress:    HostMachineIPAddress,
	}
}

type LoadBalancerInfo struct {
	HealthInfo *LoadBalancerHealthInfo `json:"HealthInfo,omitempty;" xml:"HealthInfo"`
	IPAddress  string                  `json:"IPAddress" xml:"IPAddress"`
}

func NewLoadBalancerInfo(HealthInfo LoadBalancerHealthInfo, IPAddress string) *LoadBalancerInfo {
	return &LoadBalancerInfo{
		HealthInfo: &HealthInfo,
		IPAddress:  IPAddress,
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

type LoadBalancer struct {
	BaseLoadBalancer
}

func NewLoadBalancer(LoadBalancerParams *InitializationParams) *LoadBalancer {
	return &LoadBalancer{}
}

func (this *LoadBalancer) Create() (*LoadBalancerInfo, error) {
	// Initializes New Load Balancer
}

func (this *LoadBalancer) Delete(LoadBalancerIPAddress string) (bool, error) {
	// Deletes Existing Load Balancer
}

func (this *LoadBalancer) GetInfo(LoadBalancerIPAddress string) (*LoadBalancerInfo, error) {
	// Returns Load Balancer Info
}

func (this *LoadBalancer) GetHealthInfo(LoadBalancerIPAddress string) (*LoadBalancerHealthInfo, error) {
	// Returns Health Info about the Load Balancer
}

type LoadBalancerManager struct {
	BaseLoadBalancerManager
}

func NewLoadBalancerManager() *LoadBalancerManager {
	return &LoadBalancerManager{}
}

func (this *LoadBalancerManager) CreateLoadBalancer(InitParams InitializationParams) (LoadBalancerInfo, error)

func (this *LoadBalancerManager) DeleteLoadBalancer(LoadBalanceIPAddress string) (bool, error)

func (this *LoadBalancerManager) GetInfo(LoadBalancerIPAddress string) (LoadBalancerInfo, error)

func (this *LoadBalancerManager) GetHealthInfo(LoadBalancerIPAddress string) (LoadBalancerHealthInfo, error)
