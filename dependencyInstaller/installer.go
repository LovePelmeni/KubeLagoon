package installer

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/LovePelmeni/Infrastructure/models"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("Installer.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if Error != nil {
		panic(Error)
	}
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Lshortfile|log.Ltime)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Lshortfile|log.Ltime)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Lshortfile|log.Ltime)
}

type OSDeploymentToolsInstallCommandReturnerInterface interface {
	// Class, that returns specific commands to install docker, docker-compose, podman ....
	// Depending on the Operational System Specified
	GetCommands(DistributionName string, Version ...string)
	GetDockerCommand() string
	GetDockerComposeCommand() string
	GetPodmanCommand() string
	GetVirtualBoxCommand() string
}

type WindowsDeploymentToolsInstallCommandReturner struct {
	OSDeploymentToolsInstallCommandReturnerInterface

	Installers map[string]func(DistributionName string, Version ...string) string
}

func NewWindowsDeploymentToolsinstallCommandReturner() *WindowsDeploymentToolsInstallCommandReturner {
	var CommandReturner WindowsDeploymentToolsInstallCommandReturner

	return &WindowsDeploymentToolsInstallCommandReturner{
		Installers: map[string]func(DistributionName string, Version ...string) string{
			"Docker":         CommandReturner.GetDockerCommand(),
			"Docker-Compose": CommandReturner.GetDockerComposeCommand(),
			"Podman":         CommandReturner.GetPodmanCommand(),
			"VirtualBox":     CommandReturner.GetVirtualBoxCommand(),
		},
	}
}

func (this *WindowsDeploymentToolsInstallCommandReturner) GetDockerCommand() string {
	// Returns Command for the Installation Module, (for the Windows OS), Also Depending on the Version
}
func (this *WindowsDeploymentToolsInstallCommandReturner) GetDockerComposeCommand() string {
	// Returns Command for the Installation Module, (for the Windows OS), Also Depending on the Version
}
func (this *WindowsDeploymentToolsInstallCommandReturner) GetPodmanCommand() string {
	// Returns Command for the Installation Module, (for the Windows OS), Also Depending on the Version
}
func (this *WindowsDeploymentToolsInstallCommandReturner) GetVirtualBoxCommand() string {
	// Returns Command for the Installation Module, (for the Windows OS), Also Depending on the Version
}

type LinuxDeploymentToolsInstallCommandReturner struct {
	OSDeploymentToolsInstallCommandReturnerInterface

	// Class for Installing Tools on the Linux Based OS
	// Is Getting Chosed if the Customer has VM Server with the Linux OS Running on top of it

	// ToolName & Method, that Returns Required Installation Commands for that Tool
	Installers map[string]func(DistributionName string, Version ...string) string
}

func NewLinuxDeploymentToolsInstallCommandReturner() *LinuxDeploymentToolsInstallCommandReturner {
	return &LinuxDeploymentToolsInstallCommandReturner{}
}

func (this *LinuxDeploymentToolsInstallCommandReturner) GetInstallationCommands(ToolNames []string, DistributionName string, Version string) []string {
	// Returns List of the Installation Commands for the Tools Specified, for the Linux Based Systems such as Ubuntu, Debian
	var ChosenTools []string
	var Commands []string
	Tools := []string{"Docker", "Docker-Compose", "Podman", "VirtualBox"}
	for _, Tool := range ToolNames {
		if slices.Contains(Tools, strings.ToLower(Tool)) {
			ChosenTools = append(ChosenTools, strings.ToLower(Tool))
		}
	}
	for _, Tool := range ChosenTools {
		if slices.Contains(maps.Keys(this.Installers), Tool) {
			Command := this.Installers[Tool](DistributionName, Version)
			Commands = append(Commands, Command)
		}
	}
	return Commands
}

func (this *LinuxDeploymentToolsInstallCommandReturner) GetDockerCommand(DistributionName string, Version ...string) string {
	// Returns Docker Installation Command for Linux Based OS
	return fmt.Sprintf("RUN curl -fsSL https://download.docker.com/linux/%s/gpg | apt-key add - && ", strings.ToLower(DistributionName)) +
		"RUN add-apt-repository \\ " +
		fmt.Sprintf("deb [arch=amd64] https://download.docker.com/linux/%s \\ ", strings.ToLower(DistributionName)) +
		" $(lsb_release -cs) \\ " +
		"stable && " + "docker --version"
}

func (this *LinuxDeploymentToolsInstallCommandReturner) GetDockerComposeCommand(DistributionName string, Version ...string) string {
	// Returns Docker-Compose Installation Command for the Linux Based OS
	return "RUN curl -L 'https://github.com/docker/compose/releases/download/v2.1.1/docker-compose-$(uname -s)-$(uname -m)' -o /usr/local/bin/docker-compose && " +
		"RUN chmod +x /usr/local/bin/docker-compose && " +
		"RUN curl https://get.docker.com/ > dockerinstall && chmod 777 dockerinstall && ./dockerinstall && " + "docker-compose --version"
}

func (this *LinuxDeploymentToolsInstallCommandReturner) GetPodmanCommand(DistributionName string, Version ...string) string {
	// Returns Podman Installation Command for the Linux Based OS
	// Supports
	return "sudo apt update && " + "sudo apt install -y podman &&" +
		"sudo sh -c 'echo" +
		fmt.Sprintf("'deb http://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/x%s_%s/ /' > /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list''", strings.ToTitle(DistributionName), Version) +
		fmt.Sprintf("wget -nv https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable/x%s_%s/Release.key -O- | sudo apt-key add -", DistributionName, Version) +
		"sudo apt update && " + "sudo apt -y install podman &&" + " podman --version"
}

func (this *LinuxDeploymentToolsInstallCommandReturner) GetVirtualBoxCommand(DistributionName string, Version ...string) string {
	// Returns VirtualBox Installation Command for the
	return ""
}

type Dependency struct {
	PackageName string   `json:"PackageName"`
	InstallUrl  *url.URL `json:"InstallUrl"`
}

func (this *Dependency) UploadToVm(DependencyCommands []string, SshConnection ssh.Client) string {
	// Uploads package to the Virtual Machine, Returns Output of the Command
	NewSshSession, SshError := SshConnection.NewSession()
	if SshError != nil {
		ErrorLogger.Printf(
			"Failed to Start new SSH Session to Remote Virtual Machine, Error: %s", SshError)
		return "ERROR"
	}
	defer NewSshSession.Close()

	var StdoutBuffer bytes.Buffer
	var CommandError error
	NewSshSession.Stdout = &StdoutBuffer

	for _, Command := range DependencyCommands {

		if len(Command) != 0 {
			ExecutionError := NewSshSession.Run(
				Command)
			CommandError = ExecutionError
		}

		if CommandError != nil {
			ErrorLogger.Printf(
				"Failed to Execute SSH Command, Error: %s",
				CommandError)
			return "ERROR"
		}
	}
	return StdoutBuffer.String()
}

func NewDependency(PackageName string, InstallUrl url.URL) *Dependency {
	return &Dependency{
		PackageName: PackageName,
		InstallUrl:  &InstallUrl,
	}
}

type EnvironmentDependencyInstallerInterface interface {
	// Interface, represents Dependency Installer, that Allows to PreInstall
	// Packages, Interpreters and Other Soft to the Virtual Machine
	GetSshConnection(PublicKey []byte, VmIP string) (ssh.Conn, error)
	InstallDependencies(Dependencies []Dependency) bool
	GetDependency(PackageName string, InstallUrl string) bool
}

type EnviromentDependencyInstaller struct{}

func NewEnviromentDependencyInstaller() *EnviromentDependencyInstaller {
	return &EnviromentDependencyInstaller{}
}

func (this *EnviromentDependencyInstaller) GetSshConnection(VirtualMachineId string) (*ssh.Client, error) {
	// Returns SSH Connection to the VM Server
	var VirtualMachine models.VirtualMachine
	models.Database.Model(&models.VirtualMachine{}).Where("id = ?", VirtualMachineId).Find(&VirtualMachine)
	ClientConfig := &ssh.ClientConfig{
		Timeout: 10,
		User:    "root",
		Auth:    []ssh.AuthMethod{},
	}
	NewSshConnection, ConnectionError := ssh.Dial("TCP", VirtualMachine.IPAddress, ClientConfig)
	return NewSshConnection, ConnectionError
}

func (this *EnviromentDependencyInstaller) GetDependency(PackageName string, InstallUrl string) (*Dependency, error) {
	// Returns Dependency Structure of the Dependency
	return NewDependency(PackageName, url.URL{Path: InstallUrl}), nil
}

func (this *EnviromentDependencyInstaller) InstallDeploymentDependencies(SshConnection ssh.Client) error {
	// Installs deployment Dependencies such as Docker and Docker-Compose and Kubectl
	var Responses []string
	var stdOut bytes.Buffer

	NewSession, SSHError := SshConnection.NewSession()
	if SSHError != nil {
		return SSHError
	}
	NewSession.Stdout = &stdOut

	InstallationCommands := []string{}

	for _, Command := range InstallationCommands {
		ResponseError := NewSession.Run(Command)
		if len(stdOut.String()) != 0 {
			Responses = append(Responses)
		}
		if strings.Contains(stdOut.String(), "error") || ResponseError != nil {
			return errors.New(ResponseError.Error())
		}
	}
	return nil
}
