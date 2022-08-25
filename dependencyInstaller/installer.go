package installer

import (
	"bytes"
	"errors"
	"log"
	"net/url"
	"os"
	"strings"
	"github.com/LovePelmeni/Infrastructure/models"
	"golang.org/x/crypto/ssh"
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


type EnviromentDependencyInstaller struct {}

func NewEnviromentDependencyInstaller() *EnviromentDependencyInstaller {
	return &EnviromentDependencyInstaller{}
}

func (this *EnviromentDependencyInstaller) GetSshConnection(VirtualMachineId string) (*ssh.Client, error){
	// Returns SSH Connection to the VM Server 
	var VirtualMachine models.VirtualMachine
	models.Database.Model(&models.VirtualMachine{}).Where("id = ?", VirtualMachineId).Find(&VirtualMachine)
	ClientConfig := &ssh.ClientConfig{
		Timeout: 10,
		User: "root", 
		Auth: []ssh.AuthMethod{},
	}
	NewSshConnection, ConnectionError := ssh.Dial("TCP", VirtualMachine.IPAddress, ClientConfig)
	return NewSshConnection, ConnectionError
}

func (this *EnviromentDependencyInstaller) GetDependency(PackageName string, InstallUrl string) (*Dependency, error) {
	// Returns Dependency Structure of the Dependency 
	return NewDependency(PackageName, url.URL{Path: InstallUrl}), nil
}

func (this *EnviromentDependencyInstaller) GetLinuxDeploymentCommand()

func (this *EnviromentDependencyInstaller) GetWindowsDeploymentCommand()

func (this *EnviromentDependencyInstaller) GetCentosDeploymentCommand()


func (this *EnviromentDependencyInstaller) InstallDeploymentDependencies(SshConnection ssh.Client) error {
	// Installs deployment Dependencies such as Docker and Docker-Compose and Kubectl
	var InstallationCommands = []string{
		"curl -fsSL https://get.docker.com -o get-docker.sh" +
		"$ DRY_RUN=1 sh ./get-docker.sh", 
		"usermod -aG docker root",
	}
	var Responses []string 
	NewSession, SSHError := SshConnection.NewSession()
	if SSHError != nil {return SSHError}
	for _, Command := range InstallationCommands {
		var stdOut bytes.Buffer 
		ResponseError := NewSession.Run(Command)
		if len(stdOut.String()) != 0 {Responses = append(Responses, )}
		if strings.Contains(stdOut.String(), "error") || ResponseError != nil {
		return errors.New(ResponseError.Error())}
	}
}
