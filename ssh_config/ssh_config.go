package ssh_config

import (
	"context"
	"errors"

	"fmt"
	"io"

	"log"
	"net/url"

	"os"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"

	"github.com/vmware/govmomi/vapi/appliance/access/ssh"
	"github.com/vmware/govmomi/vapi/rest"

	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
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

type PublicKey struct {
	FilePath string `json:"FilePath" xml:"FilePath"`
	FileName string `json:"FileName" xml:"FileName"`
	Content  []byte `json:"Content" xml:"Content"`
}

func NewPublicKey(Content []byte, FileName string) *PublicKey {
	return &PublicKey{
		FileName: FileName,
		Content:  Content,
	}
}

type PrivateKey struct {
	FileName string `json:"FileName" xml:"FileName"`
	Content  []byte `json:"Content" xml:"Content"`
}

func NewPrivateKey(Content []byte, FileName string) *PrivateKey {
	return &PrivateKey{
		Content: Content,
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

func NewVirtualMachineSshManager(Client vim25.Client, VirtualMachine *object.VirtualMachine) *VirtualMachineSshCertificateManager {
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
		Path: "/"}, nil
}

func (this *VirtualMachineSshCertificateManager) UploadSshKeys(Key PrivateKey) error {
	// Uploaded SSH Pem Key to the Virtual Machine Server...

	SshManager := ssh.NewManager(rest.NewClient(&this.Client))
	VirtualMachineFullUrl, UrlError := this.GetVirtualMachineUrl()
	if UrlError != nil {
		ErrorLogger.Printf("Failed to Parse Virtual Machine Url")
		return UrlError
	}

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	PrivateUploadedError := SshManager.UploadFile(TimeoutContext, Key.FileName, VirtualMachineFullUrl, nil)
	if PrivateUploadedError != nil {
		ErrorLogger.Printf(
			"Failed to Upload Private SSH Key to the Virtual Machine, Error: %s", PrivateUploadedError)
		return PrivateUploadedError
	}

	Access := ssh.Access{Enabled: true}

	// Setting up PEM Paths....

	PEMPathSetupError := SshManager.SetRootCAs(this.GetDefaultPEMPath())
	if PEMPathSetupError != nil {
		ErrorLogger.Printf("Failed to Setup Default Private Key Paths, Error: %s", PEMPathSetupError)
		return PEMPathSetupError
	}

	if State, Error := SshManager.Get(TimeoutContext); State != true && Error == nil {
		// If Ssh CLI is not enabled, Enabling It....
		SetAccessError := SshManager.Set(TimeoutContext, Access)
		if SetAccessError != nil {
			ErrorLogger.Printf("Failed to Enable SSH CLI, Error: %s", SetAccessError)
		}
	}
	return nil
}

func (this *VirtualMachineSshCertificateManager) GenerateSshKeys() (*PublicKey, *PrivateKey, error) {
	// Returns Generated SSH Keys for the Virtual Machine Server

	Manager := ssh.NewManager(rest.NewClient(&this.Client))
	GeneratedCertificate := Manager.Certificate()
	PrivateKey := GeneratedCertificate.Leaf.Raw
	PublicKey := Manager.Certificate().Leaf.RawSubjectPublicKeyInfo

	var GenerationError error

	// Writing Private Key to the Temporary Buffer
	PrivateKeyWriter := io.MultiWriter()
	PrivateKeyWriter.Write(PrivateKey)

	// Writing Public Key to the Temporary Buffer
	PublicKeyWriter := io.MultiWriter()
	PublicKeyWriter.Write(PublicKey)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	// Writing New SSH Key Files for Public and Private One 
	NewPrivateKeyFileError := Manager.WriteFile(TimeoutContext, fmt.Sprintf("%s_ssh_key.pem", this.VirtualMachine.Name()), io.MultiReader(), 2048, nil, PrivateKeyWriter)
	NewPublicKeyFileError := Manager.WriteFile(TimeoutContext, fmt.Sprintf("%s_ssh_key.pub", this.VirtualMachine.Name()), io.MultiReader(), 2048, nil, PublicKeyWriter)

	if NewPrivateKeyFileError != nil || NewPublicKeyFileError != nil {
		GenerationError = errors.New("Failed to Generate SSH Keys")
	}
	return NewPublicKey(PublicKey, "ssh_key.pub"), NewPrivateKey(PrivateKey, "ssh_key.pem"), GenerationError
}


type VirtualMachineSshRootCredentialsManager struct {
	// SSH Manager Class, that performs Type of the SSH Connection 
	// Via Root Credentials
	VirtualMachineSshManager
	Client vim25.Client 
	VirtualMachine object.VirtualMachine 
}

func NewVirtualMachineRootCredentialsManager(Client vim25.Client, VirtualMachine *object.VirtualMachine) *VirtualMachineSshRootCredentialsManager {
	return &VirtualMachineSshRootCredentialsManager{
		Client: Client, 
		VirtualMachine: *VirtualMachine,
	}
}
func (this *VirtualMachineSshRootCredentialsManager) GetSshRootCredentials() (*SshRootCredentials, error){
	// Parses Root Credentials of the OS Host System of the Customer's Virtual Machine Server 
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	Manager := property.DefaultCollector(&this.Client)
	defer CancelFunc()

	// Receiving Virtual Machine Instance 

	var VirtualMachine mo.VirtualMachine 
	RetrieveError := Manager.RetrieveOne(TimeoutContext, this.VirtualMachine.Reference(),
    []string{"name", "guest"}, &VirtualMachine)

	if RetrieveError != nil {DebugLogger.Printf(
	"Failed to Get VirtualMachine Instance"); return nil, RetrieveError}
}