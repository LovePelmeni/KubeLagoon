package ip

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

type VirtualMachineIPManagerInterface interface {
	// Interface of the Class, that setting Up IP Address to the Virtual Machine
	SetupAddress(IPCredentials *VirtualMachineIPAddress) (*VirtualMachineIPAddress, error)
}

type VirtualMachineIPManager struct {
	VirtualMachineIPManagerInterface
}

func NewVirtualMachineIPManager() *VirtualMachineIPManager {
	return &VirtualMachineIPManager{}
}

func (this *VirtualMachineIPManager) SetupAddress(IPCredentials *VirtualMachineIPAddress) (*types.CustomizationSpec, error) {

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
		NicSettingMap: []types.CustomizationAdapterMapping{CustomizedIP}, // Adding Previous Configuration
		Identity: &types.CustomizationLinuxPrep{
			HostName: &types.CustomizationFixedName{Name: IPCredentials.Hostname}, // Setting up Identity Hostname
		}}

	return CustomizedIPSettings, nil
}
