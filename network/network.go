package network 

import (
	"log"
	"os"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("IP.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

type VirtualMachineIPAddress struct {
	// Struct, Representing Virtual Machine IP Address 
	Options types.BaseCustomizationOptions 
	IP       string `json:"IP"`
	Netmask  string `json:"Netmask"`
	Gateway  string `json:"Gateway"`
	Hostname string `json:"Hostname"`
}

func NewVirtualMachineIPAddress(IP string, Netmask string, Gateway string, Hostname string) *VirtualMachineIPAddress {
	return &VirtualMachineIPAddress{
		IP:       IP,
		Netmask:  Netmask,
		Gateway:  Gateway,
		Hostname: Hostname,
	}
}

type VirtualMachineIPManager struct{}

// Virtual Machine IP Manager Class 

func NewVirtualMachineIPManager() *VirtualMachineIPManager {
	return &VirtualMachineIPManager{}
}

func (this *VirtualMachineIPManager) SetupPublicNetwork(IPCredentials *VirtualMachineIPAddress) (*types.CustomizationSpec, error) {

	// Setting up Customized IP Credentials for the Virtual Machine
	CustomizedIP := types.CustomizationAdapterMapping{
		Adapter: types.CustomizationIPSettings{

			Ip:         &types.CustomizationFixedIp{IpAddress: IPCredentials.IP}, // Setting UP IP Address
			SubnetMask: IPCredentials.Netmask,                                    // Setting UP Subnet Mask
			Gateway:    []string{IPCredentials.Gateway},                          // Setting up Gateway
		},
	}
	// Updating Customized IP Setting Configuration with the Previous IP Configuration
	CustomizedIPSettings := &types.CustomizationSpec{
		Options: IPCredentials.Options, 
		NicSettingMap: []types.CustomizationAdapterMapping{CustomizedIP}, // Adding Previous Configuration
		Identity: &types.CustomizationLinuxPrep{
			HostName: &types.CustomizationFixedName{Name: IPCredentials.Hostname}, // Setting up Identity Hostname
		}}

	return CustomizedIPSettings, nil
}

type PrivateNetworkCredentials struct {
	EnableIPv6 bool 
	EnableIPv4 bool 
	SubnetAddr string 
	SubNetMask string 
}

func NewPrivateNetworkCredentials(Enablev6 bool, Enablev4 bool, Netmask string, SubnetAddr string) *PrivateNetworkCredentials {
	return &PrivateNetworkCredentials{
		EnableIPv6: Enablev6,
		EnableIPv4: Enablev4,
		SubnetAddr: SubnetAddr,
		SubNetMask: Netmask,
	}
}

// type VirtualMachinePrivateNetworkManager struct {}

// func NewVirtualMachinePrivateNetworkManager() *VirtualMachinePrivateNetworkManager {
// 	return &VirtualMachinePrivateNetworkManager{}
// }

// func (this *VirtualMachinePrivateNetworkManager) SetupPrivateNetwork(IPCredentials PrivateNetworkCredentials) {
// 	// Returns Configuration of the Private Network, based on the Customization Parameters 

// 	PrivateHostSpec := types.HostDhcpServiceSpec{
// 		IpSubnetAddr: IPCredentials.SubnetAddr,  
// 		IpSubnetMask: IPCredentials.SubNetMask,
// 	}

// 	PrivateNetworkSpec := types.NetDhcpConfigSpec{
// 		Ipv6: &types.NetDhcpConfigSpecDhcpOptionsSpec{
// 			Enable: types.NewBool(IPCredentials.EnableIPv6),
// 		}, 
// 		Ipv4: &types.NetDhcpConfigSpecDhcpOptionsSpec{
// 			Enable: types.NewBool(IPCredentials.EnableIPv4),
// 		},
// 	}
// 	PrivateHostService := types.HostDhcpService{
// 		Spec: PrivateHostSpec,
// 	}
// 	PrivateNetworkService := types.HostService
// }