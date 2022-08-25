package network

import (
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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
	IP       string `json:"IP,omitempty"`
	Netmask  string `json:"Netmask,omitempty"`
	Gateway  string `json:"Gateway,omitempty"`
	Hostname string `json:"Hostname,omitempty"`
}

func (this *VirtualMachineIPAddress) GetValidationRegexPatterns() map[string]string{
	// returns Slice of the Regexes 
	return map[string]string{}
}

func (this *VirtualMachineIPAddress) ValidateCredentials() VirtualMachineIPAddress{

	// Checks if the Input has appropriate format and has valid values 
	var InvalidValues []string // array of the Invalid Value Field names
	FieldValueGenerators := map[string]func() types.CustomizationDhcpIpGenerator {
		
		"Gateway": func() types.CustomizationDhcpIpGenerator {
			return types.CustomizationDhcpIpGenerator{}
		},
		"Netmask": func() types.CustomizationDhcpIpGenerator {
			return types.CustomizationDhcpIpGenerator{}
		},
		"Hostname": func() types.CustomizationDhcpIpGenerator {
			return types.CustomizationDhcpIpGenerator{}
		},
	}

	//  Validating Inputs 
	Patterns := this.GetValidationRegexPatterns()
	for Index := 0; Index < reflect.TypeOf(this).NumField(); Index ++ {
		if Matches, MatchError := regexp.MatchString(Patterns[strings.ToLower(reflect.ValueOf(this).Type().Field(Index).Name)],
	    reflect.ValueOf(this).Field(Index).String()); MatchError != nil || Matches != true {
			InvalidValues = append(InvalidValues, reflect.ValueOf(this).Type().Field(Index).Name)
		}
	}

	// Generating new Values if Some of the Are Empty 
	for _, Field := range InvalidValues {
		if slices.Contains(maps.Keys(FieldValueGenerators), strings.ToTitle(Field)){ 
			GeneratedValue := FieldValueGenerators[Field]()
			reflect.ValueOf(this).FieldByName(Field).Set(reflect.ValueOf(GeneratedValue))
		}
	}
	return *this
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

func (this *VirtualMachineIPManager) SetupPublicNetwork(IPCredentials VirtualMachineIPAddress) (*types.CustomizationSpec, error) {

	IPCredentials = IPCredentials.ValidateCredentials()
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