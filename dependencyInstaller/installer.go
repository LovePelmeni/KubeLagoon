package installer

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/LovePelmeni/Infrastructure/host_system"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"

	"github.com/vmware/govmomi/vim25/mo"
	"golang.org/x/crypto/ssh"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	DebugLogger *log.Logger 
	InfoLogger *log.Logger 
	ErrorLogger *log.Logger 
)

func init() {
	LogFile, Error := os.OpenFile("Installer.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if Error != nil {panic(Error)}
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Lshortfile|log.Ltime)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Lshortfile|log.Ltime)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Lshortfile|log.Ltime)
}

type InstallCommandFormer struct {
	OperationalSystem string // linux, windows, centos etc.. 
}

func NewInstallCommandFormer(OperationalSystem string) *InstallCommandFormer {
	return &InstallCommandFormer{
		OperationalSystem: OperationalSystem,
	}
}

func (this *InstallCommandFormer) GetCommands(Dependency Dependency) ([]string, error) {
	// Returns Installation Sequence of the Commands for the Specific Package, depending on the OS 

	if strings.Contains(strings.ToLower(this.OperationalSystem), "linux") {
		return this.GetLinuxInstallCommands(Dependency.PackageName, Dependency.InstallUrl), nil
	}
	if strings.Contains(strings.ToLower(this.OperationalSystem), "centos") {
		return this.GetCentosInstallCommands(Dependency.PackageName, Dependency.InstallUrl), nil
	}
	if strings.Contains(strings.ToLower(this.OperationalSystem), "win") {
		return this.GetWindowsInstallCommands(Dependency.PackageName, Dependency.InstallUrl), nil
	}
	if strings.Contains(strings.ToLower(this.OperationalSystem), "fedora") {
		return this.GetFedoraInstallCommands(Dependency.PackageName, Dependency.InstallUrl), nil
	}
	return nil, errors.New("Unsupported OS")
}

func (this *InstallCommandFormer) GetLinuxInstallCommands(PackageName string, PackageUrl *url.URL) []string{
	// Returns Installation Package Command of the Linux
	return []string{}
}
func (this *InstallCommandFormer) GetCentosInstallCommands(PackageName string, PackageUrl *url.URL) []string {
	// Returns Installation Package Command for Centos 
	return []string{}
}
func (this *InstallCommandFormer) GetFedoraInstallCommands(PackageName string, PackageUrl *url.URL) []string {
	// Returns Installation Package Command for Centos 
	return []string{}
}
func (this *InstallCommandFormer) GetWindowsInstallCommands(PackageName string, PackageUrl *url.URL) []string {
	// Returns Installation Package Command for Centos 
	return []string{}
}

type Dependency struct {
	PackageName string `json:"PackageName"`
	InstallUrl  *url.URL`json:"InstallUrl"`
}

func (this *Dependency) UploadToVm(DependencyCommands []string, SshConnection ssh.Client) string {
	// Uploads package to the Virtual Machine, Returns Output of the Command
	NewSshSession, SshError := SshConnection.NewSession()
	if SshError != nil {ErrorLogger.Printf(
	"Failed to Start new SSH Session to Remote Virtual Machine, Error: %s", SshError); return "ERROR"}
	defer NewSshSession.Close()

	var StdoutBuffer bytes.Buffer
	var CommandError error 
	NewSshSession.Stdout = &StdoutBuffer

	for _, Command := range DependencyCommands {
		
		if len(Command) != 0 {ExecutionError := NewSshSession.Run(
		Command); CommandError = ExecutionError}

		if CommandError != nil {ErrorLogger.Printf(
		"Failed to Execute SSH Command, Error: %s",
	    CommandError); return "ERROR"}
	}
	return StdoutBuffer.String()
}

func NewDependency(PackageName string, InstallUrl url.URL) *Dependency {
	return &Dependency{
		PackageName: PackageName,
		InstallUrl: &InstallUrl,
	}
}

type EnvironmentDependencyInstallerInterface interface {
	// Interface, represents Dependency Installer, that Allows to PreInstall 
	// Packages, Interpreters and Other Soft to the Virtual Machine 
	GetSshConnection(PublicKey []byte, VmIP string) (ssh.Conn, error)
	InstallDependencies(Dependencies []Dependency) bool 
	GetDependency(PackageName string, InstallUrl string) bool 
}


type EnviromentDependencyInstaller struct {
	VirtualMachine object.VirtualMachine
}

func (this *EnviromentDependencyInstaller) GetSshConnection() (*ssh.Client, error){
	// Returns SSH Connection to the VM Server 
	ClientConfig := &ssh.Config{

	}
	var MoVirtualMachine mo.VirtualMachine
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	RetrieveError := property.DefaultCollector(this.VirtualMachine.Client()).RetrieveOne(
	TimeoutContext, this.VirtualMachine.Reference(), []string{"guest"}, &MoVirtualMachine)

	VirtualMachineIPAddress := MoVirtualMachine.Config.ToConfigSpec().VAppConfig.GetVmConfigSpec().IpAssignment

	NewSshConnection, ConnectionError := ssh.Dial("TCP", VirtualMachineIPAddress, ClientConfig)
	return NewSshConnection, ConnectionError
}

func (this *EnviromentDependencyInstaller) GetDependency(PackageName string, InstallUrl string) (*Dependency, error) {
	// Returns Dependency Structure of the Dependency 
	return NewDependency(PackageName, url.URL{Path: InstallUrl}), nil
}

func (this *EnviromentDependencyInstaller) InstallDependencies(Dependencies []Dependency) (int, int, error) {

	// Installes all Dependencies that has been Provided by the Customer to the Virtual Machine 

	SSHServerConnection, SSHError := this.GetSshConnection()
	if SSHError != nil {return 0, 0, SSHError}

	var MoVirtualMachine mo.VirtualMachine
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	RetrieveError := property.DefaultCollector(this.VirtualMachine.Client()).RetrieveOne(
	TimeoutContext, this.VirtualMachine.Reference(), []string{"guest"}, &MoVirtualMachine)

	if RetrieveError != nil {return 0, 0, RetrieveError}

	//Checking Operational System of the VM Server 

	LinuxDistributions := maps.Keys(host_system.LinuxDistributions)
	if !slices.Contains(LinuxDistributions, strings.ToLower(MoVirtualMachine.Guest.GuestId)) {
		return 0, 0, errors.New("Unsupported Operational System")
	}

	var SuccessInstallPackages int
	var FailedInstallPackages int

	CommandManager := NewInstallCommandFormer(this.VirtualMachine)
	for _, Dependency := range Dependencies {

		DependencyCommands := CommandManager.GetCommands(Dependency)

		if InstalledOutput := Dependency.UploadToVm(
		DependencyCommands, *SSHServerConnection); InstalledOutput == "ERROR" {
			ErrorLogger.Printf("Failed to Install Dependency with Name: %s, Error: %s", 
			Dependency.PackageName, InstalledOutput); 
			FailedInstallPackages ++; continue 
		}else{
			SuccessInstallPackages ++
		}
	}
	return SuccessInstallPackages, FailedInstallPackages, nil
}