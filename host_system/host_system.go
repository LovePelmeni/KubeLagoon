package host_system

import (
	"errors"
	"strconv"
	"strings"

	"github.com/vmware/govmomi/object"
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
	HostSystem  *object.HostSystem `json:"HostSystem"`
	VmIPAddress string             `json:"IPAddress"`
	Hostname    string             `json:"Hostname"`
}

func NewHostSystemCredentials(SystemName string, Bit int64) *HostSystemCredentials {

	return &HostSystemCredentials{
		SystemName: strings.ToLower(SystemName),
		Bit:        Bit,
	}
}

type VirtualMachineHostSystemManager struct{}

func NewVirtualMachineHostSystemManager() *VirtualMachineHostSystemManager {
	return &VirtualMachineHostSystemManager{}
}

func (this *VirtualMachineHostSystemManager) SelectLinuxHostSystemGuest(DistributionName string, Bit ...int64) (*types.VirtualMachineGuestOsIdentifier, error) {
	// Picking up const for the Operational System User, depending on the Linux Distribution
	// Currently Supported: Ubuntu64, Debian6, Debian7, Debian8, Debian9, Fedora, Asianux, CentOS, Freebsd
	Oss := map[string]types.VirtualMachineGuestOsIdentifier{
		"ubuntu64":    types.VirtualMachineGuestOsIdentifierUbuntu64Guest,
		"ubuntu":      types.VirtualMachineGuestOsIdentifierUbuntuGuest,
		"debian10_64": types.VirtualMachineGuestOsIdentifierDebian10_64Guest,
		"debian10":    types.VirtualMachineGuestOsIdentifierDebian10Guest,
		"centos64":    types.VirtualMachineGuestOsIdentifierCentos64Guest,
		"centos":      types.VirtualMachineGuestOsIdentifierCentosGuest,
		"fedora":      types.VirtualMachineGuestOsIdentifierFedoraGuest,
		"fedora64":    types.VirtualMachineGuestOsIdentifierFedora64Guest,
	}

	for OsName, Identifier := range Oss {
		if HasPrefix := strings.HasPrefix(strings.ToLower(OsName),
			strings.ToLower(DistributionName)) && strings.HasSuffix(OsName, strconv.Itoa(int(Bit[0]))); HasPrefix != false {
			return &Identifier, nil
		} else {
			continue
		}
	}
	return nil, errors.New("Unsupported Operational System has been Specified")
}

func (this *VirtualMachineHostSystemManager) GetDefaultCustomizationOptions(SystemName string) (types.BaseCustomizationOptions, error) {
	// Returns Customization Options, based on the Operational System passed

	// Returning Linux Customization Options, if the Operational System for the VM is Linux Distribution
	if Contains := slices.Contains(maps.Keys(LinuxDistributions), strings.ToLower(SystemName)); Contains {
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

	DefaultCustomizationOptions, OptionsError := this.GetDefaultCustomizationOptions(HostSystemCredentials.SystemName)
	if OptionsError != nil {
		return nil, nil, OptionsError
	}
	HostSystemCustomizationConfig := types.CustomizationSpec{
		Options: DefaultCustomizationOptions,
	}
	OSGuest, SelectError := this.SelectLinuxHostSystemGuest(HostSystemCredentials.SystemName)
	if SelectError != nil {
		return nil, nil, SelectError
	}

	VmGuestSummaryConfig := types.VirtualMachineGuestSummary{
		GuestId:   string(*OSGuest),
		HostName:  HostSystemCredentials.Hostname,
		IpAddress: HostSystemCredentials.VmIPAddress,
	}
	return &VmGuestSummaryConfig, &HostSystemCustomizationConfig, nil
}
