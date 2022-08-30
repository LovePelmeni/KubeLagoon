package ssh_config

import (
	"context"
	"errors"

	"fmt"

	"log"
	"net/url"

	"os"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"

	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("Ssh.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if Error != nil {
		panic(Error)
	}
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ltime|log.Ldate|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ltime|log.Ldate|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ltime|log.Ldate|log.Lshortfile)
}

type SshCredentialsInterface interface {
	// Interface, represents base class, that represents SSH Credentials
}

type SshRootCredentials struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func NewSshRootCredentials(Username string, Password string) *SshRootCredentials {
	// Returns New Instance of the SSH Root Credentials
	return &SshRootCredentials{
		Username: Username,
		Password: Password,
	}
}

type SshCertificateCredentials struct {
	FilePath string `json:"FilePath" xml:"FilePath"`
	FileName string `json:"FileName" xml:"FileName"`
	Content  []byte `json:"Content" xml:"Content"`
}

func NewSshCertificateCredentials(Content []byte, FileName string) *SshCertificateCredentials {
	return &SshCertificateCredentials{
		FileName: FileName,
		Content:  Content,
	}
}

type VirtualMachineSshManager interface {
	// Interface, represents base SSH Manager Interface for the Virtual Machine Server
}

type VirtualMachineSshCertificateManager struct {
	VirtualMachineSshManager
	Client         vim25.Client
	VirtualMachine *object.VirtualMachine
}

func NewVirtualMachineSshCertificateManager(Client vim25.Client, VirtualMachine *object.VirtualMachine) *VirtualMachineSshCertificateManager {
	return &VirtualMachineSshCertificateManager{
		Client:         Client,
		VirtualMachine: VirtualMachine,
	}
}

func (this *VirtualMachineSshCertificateManager) GetDefaultPEMPath() string {
	// Returns Default SSH Path on the VM, where the SSH Keys is going to be Uploaded To
	return "/ssh-pem/"
}

func (this *VirtualMachineSshCertificateManager) GetVirtualMachineUrl() (*url.URL, error) {
	// Returns Full Virtual Machine Url
	var MoVirtualMachine mo.VirtualMachine
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	Collector := property.DefaultCollector(&this.Client)
	MoVirtualMachineError := Collector.RetrieveOne(TimeoutContext,
		this.VirtualMachine.Reference(), []string{"*"}, &MoVirtualMachine)

	if MoVirtualMachineError != nil {
		ErrorLogger.Printf(
			"Failed to Obtain Vm `MO` Version, Error: %s", MoVirtualMachineError)
	}

	return &url.URL{
		Scheme: "vmrc",
		Host:   MoVirtualMachine.Summary.Config.VmPathName,
		User: url.UserPassword(os.Getenv("VMWARE_SOURCE_USERNAME"),
			os.Getenv("VMWARE_SOURCE_PASSWORD")),

		RawQuery: fmt.Sprintf("moid=%s", this.VirtualMachine.Name()),
		Path:     "/"}, nil
}

func (this *VirtualMachineSshCertificateManager) UploadSshKeys(Key SshCertificateCredentials) error {
	// Uploaded SSH Pem Key to the Virtual Machine Server...

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	// Initializing New SSH Certificate Manager
	HostSystem, HostSystemError := this.VirtualMachine.HostSystem(TimeoutContext)
	if HostSystemError != nil {
		return errors.New("Failed to Get Host System Info")
	}
	SshManager := object.NewHostCertificateManager(&this.Client,
		*types.NewReference(this.Client.ServiceContent.RootFolder), HostSystem.Reference())

	// Uploading SSL Certificate to the Host Machine
	InstallationError := SshManager.InstallServerCertificate(TimeoutContext, string(Key.Content))
	switch InstallationError {
	case nil:
		DebugLogger.Printf("SSH Key has been Successfully Uploaded to the VM with Name: %s",
			this.VirtualMachine.Name())
		return nil

	default:
		ErrorLogger.Printf("Failed to Upload SSH Key to the Remote VM's Host Machine")
		return errors.New("Failed to Add SSH Support")
	}
}

func (this *VirtualMachineSshCertificateManager) GenerateSshKeys() (*SshCertificateCredentials, error) {

	// Returns Generated SSH Keys for the Virtual Machine Server

	// Certificate will be Generated with the Specific Name and will be Stored on the Host System
	// Of the Virtual Machine Server

	// SSL Certificate Distinguish Name Consists of the Following Pattern:
	// `VirtualMachine-<VirtualMachine's Name>`

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	HostSystem, HostSystemError := this.VirtualMachine.HostSystem(TimeoutContext)
	if HostSystemError != nil {
		return nil, errors.New("Failed to Get OS Info")
	}
	Manager := object.NewHostCertificateManager(
		&this.Client, *types.NewReference(this.Client.ServiceContent.RootFolder), HostSystem.Reference())

	SSLCertificateDistinguishName := fmt.Sprintf("VirtualMachine-%s", this.VirtualMachine.Name())
	GeneratedCertificate, GenerationError := Manager.GenerateCertificateSigningRequestByDn(TimeoutContext, SSLCertificateDistinguishName)
	return NewSshCertificateCredentials([]byte(GeneratedCertificate), "ssh_key.pub"), GenerationError
}

type VirtualMachineSshRootCredentialsManager struct {
	// SSH Manager Class, that performs Type of the SSH Connection
	// Via Root Credentials
	VirtualMachineSshManager
	Client         vim25.Client
	VirtualMachine object.VirtualMachine
}

func NewVirtualMachineSshRootCredentialsManager(Client vim25.Client, VirtualMachine *object.VirtualMachine) *VirtualMachineSshRootCredentialsManager {
	return &VirtualMachineSshRootCredentialsManager{
		Client:         Client,
		VirtualMachine: *VirtualMachine,
	}
}

func (this *VirtualMachineSshRootCredentialsManager) GetSshRootCredentials() (*types.NamePasswordAuthentication, error) {
	// Parses Root Credentials of the OS Host System of the Customer's Virtual Machine Server
	// The Returned object `types.GuestAuthentication` can be potentially used for making operations
	// that requires this authentication

	SshCredentials := types.NamePasswordAuthentication{
		Username: "",
		Password: "",
	}
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	Manager := property.DefaultCollector(&this.Client)
	defer CancelFunc()

	// Receiving Virtual Machine Instance

	var VirtualMachine mo.VirtualMachine
	RetrieveError := Manager.RetrieveOne(TimeoutContext, this.VirtualMachine.Reference(),
		[]string{"name", "guest"}, &VirtualMachine)

	if RetrieveError != nil {
		DebugLogger.Printf(
			"Failed to Get VirtualMachine Instance")
		return nil, RetrieveError
	}
	return &SshCredentials, nil
}
