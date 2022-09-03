package host_system

import (
	"errors"
	"strconv"
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"

	"github.com/vmware/govmomi/vim25/types"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var LinuxDistributions = map[string]types.VirtualMachineGuestOsIdentifier{
	"ubuntu64":    types.VirtualMachineGuestOsIdentifierUbuntu64Guest,
	"ubuntu":      types.VirtualMachineGuestOsIdentifierUbuntuGuest,
	"debian10_64": types.VirtualMachineGuestOsIdentifierDebian10_64Guest,
	"debian10":    types.VirtualMachineGuestOsIdentifierDebian10Guest,
	"centos64":    types.VirtualMachineGuestOsIdentifierCentos64Guest,
	"centos":      types.VirtualMachineGuestOsIdentifierCentosGuest,
	"fedora":      types.VirtualMachineGuestOsIdentifierFedoraGuest,
	"fedora64":    types.VirtualMachineGuestOsIdentifierFedora64Guest,
}

var WindowsDistributions = map[string]types.VirtualMachineGuestOsIdentifier{
	"windows11_64": types.VirtualMachineGuestOsIdentifierWindows11_64Guest,
	"windows9":     types.VirtualMachineGuestOsIdentifierWindows9Guest,
	"windows9_64":  types.VirtualMachineGuestOsIdentifierWindows9_64Guest,
}

// Package for Managing Host System of the Virtual Machine Server

type HostSystemCredentials struct {
	Bit         int64              `json:"Bit"`
	SystemName  string             `json:"SystemName"`
	Version     string             `json:"Version"`
	HostSystem  *object.HostSystem `json:"HostSystem"`
	VmIPAddress string             `json:"IPAddress"`
	Hostname    string             `json:"Hostname"`
}

func NewHostSystemCredentials(SystemName string, Version string, Bit ...int64) *HostSystemCredentials {

	return &HostSystemCredentials{
		SystemName: strings.ToLower(SystemName),
		Version:    Version,
		Bit:        Bit[0],
	}
}

type VirtualMachineHostSystemManager struct{}

func NewVirtualMachineHostSystemManager() *VirtualMachineHostSystemManager {
	return &VirtualMachineHostSystemManager{}
}

func (this *VirtualMachineHostSystemManager) SelectLinuxHostSystemGuest(DistributionName string, Version string, Bit ...int64) (*types.VirtualMachineGuestOsIdentifier, error) {
	// Picking up const for the Operational System User, depending on the Linux Distribution
	// Currently Supported: Ubuntu64, Debian6, Debian7, Debian8, Debian9, Fedora, Asianux, CentOS, Freebsd

	if Contains := slices.Contains(maps.Keys(LinuxDistributions),
		DistributionName+Version+"_"+strconv.Itoa(int(Bit[0]))); Contains != false {
		OperationalSystemGuest := LinuxDistributions[DistributionName+Version+"_"+strconv.Itoa(int(Bit[0]))]
		return &OperationalSystemGuest, nil
	}
	return nil, errors.New("Unsupported Operational System has been Specified")
}

func (this *VirtualMachineHostSystemManager) SelectWindowsSystemGuest(SystemName string, Version string, Bit ...int) (*types.VirtualMachineGuestOsIdentifier, error) {
	// Returning Windows Guest Interface, depending on the Distribution version of the Operational System

	for OsName, Identifier := range WindowsDistributions {
		if HasPrefix := strings.HasPrefix(strings.ToLower(OsName),
			strings.ToLower(OsName)) && strings.HasSuffix(OsName, strconv.Itoa(int(Bit[0]))); HasPrefix != false {
			return &Identifier, nil
		} else {
			continue
		}
	}
	return nil, errors.New("Unsupported Operational System has been Specified")
}

func (this *VirtualMachineHostSystemManager) GetDefaultCustomizationOptions(SystemName string, Version string, Bit int) (types.BaseCustomizationOptions, error) {
	// Returns Customization Options, based on the Operational System passed

	// Returning Linux Customization Options, if the Operational System for the VM is Linux Distribution
	if Contains := slices.Contains(maps.Keys(LinuxDistributions), strings.ToLower(SystemName+Version)+"_"+strconv.Itoa(Bit)); Contains {
		return &types.CustomizationLinuxOptions{}, nil
	}
	// Returning Windows Customization Options, if the Operational System for the VM is Windows Distribution
	if Contains := slices.Contains(maps.Keys(WindowsDistributions), strings.ToLower(SystemName)); Contains {
		return &types.CustomizationWinOptions{
			DeleteAccounts: false,
			ChangeSID:      false,
			Reboot:         types.CustomizationSysprepRebootOptionReboot,
		}, nil
	}
	return nil, errors.New("Invalid Host System Name")
}

// Returns Default Operational System Options, depending on the System Name.

func (this *VirtualMachineHostSystemManager) SetupHostSystem(HostSystemCredentials HostSystemCredentials) (*types.VirtualMachineGuestSummary, *types.CustomizationSpec, error) {

	// Returns Host Operational System based on the OS Name and Bit passed from the Customer Configuration
	DefaultCustomizationOptions, OptionsError := this.GetDefaultCustomizationOptions(HostSystemCredentials.SystemName, HostSystemCredentials.Version, int(HostSystemCredentials.Bit))
	if OptionsError != nil {
		return nil, nil, OptionsError
	}
	HostSystemCustomizationConfig := types.CustomizationSpec{
		Options: DefaultCustomizationOptions,
	}

	// Selecting Appropriate Os System Guest, Based on the Virtual Machine Host System Setup Choice
	var OSGuest *types.VirtualMachineGuestOsIdentifier
	LinuxOSGuest, SelectError := this.SelectLinuxHostSystemGuest(HostSystemCredentials.SystemName, HostSystemCredentials.Version, int64(HostSystemCredentials.Bit))
	WinOsGuest, SelectError := this.SelectWindowsSystemGuest(HostSystemCredentials.SystemName, HostSystemCredentials.Version, int(HostSystemCredentials.Bit))

	if LinuxOSGuest != nil {
		OSGuest = LinuxOSGuest
	}
	if WinOsGuest != nil {
		OSGuest = WinOsGuest
	}

	if SelectError != nil {
		return nil, nil, SelectError
	}

	// Setting up Configuration Summary

	VmGuestSummaryConfig := types.VirtualMachineGuestSummary{
		GuestId:   string(*OSGuest),
		HostName:  HostSystemCredentials.Hostname,
		IpAddress: HostSystemCredentials.VmIPAddress,
	}
	return &VmGuestSummaryConfig, &HostSystemCustomizationConfig, nil
}

func (this *VirtualMachineHostSystemManager) GetAvailableLinuxOsSystems() map[string]types.VirtualMachineGuestOsIdentifier {
	return LinuxDistributions
}

func (this *VirtualMachineHostSystemManager) GetAvailableWindowsOsSystems() map[string]types.VirtualMachineGuestOsIdentifier {
	return WindowsDistributions
}

type HostSystemNetworkManager struct {
	// Class for Managing Host System Network
	Client         vim25.Client
	VirtualMachine *object.VirtualMachine
}

func NewHostSystemNetworkManager() *HostSystemNetworkManager {
	return &HostSystemNetworkManager{}
}
