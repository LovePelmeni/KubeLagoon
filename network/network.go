package network

import (
	"context"
	"errors"
	"os"
	"time"

	"reflect"
	"regexp"

	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("NetworkLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
}

type VirtualMachineIPAddress struct {
	// Struct, Representing Virtual Machine IP Address
	Options  types.BaseCustomizationOptions
	IPv4     string `json:"IP,omitempty"`
	Netmask  string `json:"Netmask,omitempty"`
	Gateway  string `json:"Gateway,omitempty"`
	Hostname string `json:"Hostname,omitempty"`
}

func (this *VirtualMachineIPAddress) GetValidationRegexPatterns() map[string]string {
	// returns Slice of the Regexes
	return map[string]string{}
}

func (this *VirtualMachineIPAddress) ValidateCredentials() VirtualMachineIPAddress {

	// Checks if the Input has appropriate format and has valid values
	var InvalidValues []string // array of the Invalid Value Field names
	FieldValueGenerators := map[string]func() types.BaseCustomizationIpGenerator{

		"Gateway": func() types.BaseCustomizationIpGenerator {
			return &types.CustomizationCustomIpGenerator{}
		},
		"Netmask": func() types.BaseCustomizationIpGenerator {
			return &types.CustomizationDhcpIpGenerator{}
		},
		"Hostname": func() types.BaseCustomizationIpGenerator {
			return &types.CustomizationCustomIpGenerator{}
		},
	}

	//  Validating Inputs
	Patterns := this.GetValidationRegexPatterns()
	for Index := 0; Index < reflect.TypeOf(this).NumField(); Index++ {
		if Matches, MatchError := regexp.MatchString(Patterns[strings.ToLower(reflect.ValueOf(this).Type().Field(Index).Name)],
			reflect.ValueOf(this).Field(Index).String()); MatchError != nil || Matches != true {
			InvalidValues = append(InvalidValues, reflect.ValueOf(this).Type().Field(Index).Name)
		}
	}

	// Generating new Values if Some of the Are Empty
	for _, Field := range InvalidValues {
		if slices.Contains(maps.Keys(FieldValueGenerators), strings.ToTitle(Field)) {
			GeneratedValue := FieldValueGenerators[Field]()
			reflect.ValueOf(this).FieldByName(Field).Set(reflect.ValueOf(GeneratedValue))
		}
	}
	return *this
}

func NewVirtualMachineIPAddress(IPv4 string, Netmask string, Gateway string, Hostname string) *VirtualMachineIPAddress {
	return &VirtualMachineIPAddress{
		IPv4:     IPv4,
		Netmask:  Netmask,
		Gateway:  Gateway,
		Hostname: Hostname,
	}
}

type VirtualMachinePublicNetworkManager struct{}

// Virtual Machine IP Manager Class

func NewVirtualMachinePublicNetworkManager() *VirtualMachinePublicNetworkManager {
	return &VirtualMachinePublicNetworkManager{}
}

func (this *VirtualMachinePublicNetworkManager) SetupPublicNetwork(IPCredentials VirtualMachineIPAddress) (*types.CustomizationSpec, error) {

	IPCredentials = IPCredentials.ValidateCredentials()
	// Setting up Customized IP Credentials for the Virtual Machine
	CustomizedIP := types.CustomizationAdapterMapping{
		Adapter: types.CustomizationIPSettings{

			Ip:         &types.CustomizationFixedIp{IpAddress: IPCredentials.IPv4}, // Setting UP IP Address
			SubnetMask: IPCredentials.Netmask,                                      // Setting UP Subnet Mask
			Gateway:    []string{IPCredentials.Gateway},                            // Setting up Gateway
			IpV6Spec: &types.CustomizationIPSettingsIpV6AddressSpec{
				Ip: []types.BaseCustomizationIpV6Generator{
					&types.CustomizationAutoIpV6Generator{}},
			},
		},
	}
	// Updating Customized IP Setting Configuration with the Previous IP Configuration
	CustomizedIPSettings := &types.CustomizationSpec{
		Options:       IPCredentials.Options,
		NicSettingMap: []types.CustomizationAdapterMapping{CustomizedIP}, // Adding Previous Configuration
		Identity: &types.CustomizationLinuxPrep{
			HostName: &types.CustomizationFixedName{Name: IPCredentials.Hostname}, // Setting up Identity Hostname
		}}
	return CustomizedIPSettings, nil
}

func (this *VirtualMachinePublicNetworkManager) ConnectVirtualMachineToNetwork(Network *object.Network, VirtualMachine *object.VirtualMachine) {
	// Connects Virtual Machine Server to the Network specified
}

type VirtualMachinePrivateNetworkManager struct {
	// Manager For Initializing Private Network (Analogy to the VPC In the Cloud Providers)
	Client vim25.Client
}

func NewVirtualMachinePrivateNetworkManager(Client vim25.Client) *VirtualMachinePrivateNetworkManager {
	return &VirtualMachinePrivateNetworkManager{
		Client: Client,
	}
}

func (this *VirtualMachinePrivateNetworkManager) SetupPrivateNetwork(NetworkCredentials VirtualMachineIPAddress) (*object.Network, error) {
	// Returns Private Network Configuration based on the Setup that has been Required By Customer

	// Initializing Timeout Context for the Container Creation Operation
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	// Initializing Instance Manager
	Manager := view.NewManager(&this.Client)

	// Initializing New Container for the Private Network
	NewPrivateNetwork, PrivateNetworkInitializationError := Manager.CreateContainerView(
		TimeoutContext, this.Client.ServiceContent.RootFolder.Reference(), []string{"Network"}, false)
	if PrivateNetworkInitializationError != nil {
		Logger.Error(
			"Failed to Initialize New Private Network")
		return nil, errors.New("Failed to Initialize Private Network")
	}
	return object.NewReference(&this.Client,
		NewPrivateNetwork.ManagedObjectView.Reference()).(*object.Network), nil
}

func (this *VirtualMachinePrivateNetworkManager) ConnectVirtualMachineToNetwork(Network *object.Network, VirtualMachine *object.VirtualMachine) (bool, error) {
	// Connects Virtual Machine to the Private Network,
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	// Obtaining Network Resource Key

	PrivateNetworkDeviceConfig := types.VirtualDevice{
		Backing: &types.VirtualDeviceBackingInfo{},
		Connectable: &types.VirtualDeviceConnectInfo{
			Connected:         *types.NewBool(true),
			StartConnected:    *types.NewBool(true),
			AllowGuestControl: *types.NewBool(true),
		},
	}

	// Applying New Network Device, so Application can access Private Network
	ApplyError := VirtualMachine.AddDevice(TimeoutContext, &PrivateNetworkDeviceConfig)
	switch ApplyError {
	case nil:
		Logger.Error(
			"Private Network Access has been successfully added to VM: Name: %s",
			zap.String("Virtual Machine Name", VirtualMachine.Name()))
		return true, nil
	default:
		Logger.Debug(
			"Failed to Connect VM with Name: %s to the Private Network, Error: %s",
			zap.String("Virtual Machine Name", VirtualMachine.Name()), zap.Error(ApplyError))
		return false, ApplyError
	}
}

type NetworkBridgeLink struct {
	// Represents Base Network Link for the Virtual Machine Server
	VirtualMachineId   string `json:"VirtualMachineId"`
	VirtualMachineName string `json:"VirtualMachineName"`
}

func NewNetworkBridgeLink(VirtualMachineId string, VirtualMachineName string) *NetworkBridgeLink {
	return &NetworkBridgeLink{
		VirtualMachineId:   VirtualMachineId,
		VirtualMachineName: VirtualMachineName,
	}
}

type NetworkBridgeCredentials struct {
	BridgeIpAddress string `json:"BridgeIpAddress"`
}

func NewNetworkBridgeCredentials(IPAddress string) *NetworkBridgeCredentials {
	return &NetworkBridgeCredentials{
		BridgeIpAddress: IPAddress,
	}
}

type NetworkBridgeManagerInterface interface {
	// Interface implements Network Bridge Manager
	// That allows to expose the Virtual Machines to Outside Network from the Host Machine
	AddVirtualMachineLink(VirtualMachineLink string, NetworkBridgeCredentials NetworkBridgeCredentials)
	RemoveVirtualMachineLink(VirtualMachineLink string, NetworkBridgeCredentials NetworkBridgeCredentials)
}

type NetworkBridgeManager struct {
	NetworkBridgeManagerInterface
}

func NewNetworkBridgeManager() *NetworkBridgeManager {
	return &NetworkBridgeManager{}
}
func (this *NetworkBridgeManager) AddVirtualMachineLink(VirtualMachineLink NetworkBridgeLink, NetworkBridgeInfo NetworkBridgeCredentials) (bool, error) {
	// Adds the Link of the Virtual Machine Server to the Network Bridge
}

func (this *NetworkBridgeManager) RemoveVirtualMachineLink(VirtualMachineLink NetworkBridgeLink, NetworkBridgeInfo NetworkBridgeCredentials) (bool, error) {
	// Removes the Link of the Virtual Machine Server from the Network Bridge
}
