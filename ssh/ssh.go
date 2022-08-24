package ssh_config

import (
	"context"
	"errors"
	"io/fs"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/appliance/access/ssh"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
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

type PublicKey struct {
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

type VirtualMachineSshManager struct {
	Client         vim25.Client
	VirtualMachine *object.VirtualMachine
}

func (this *VirtualMachineSshManager) GetDefaultPEMPath() string {
	// Returns Default SSH Path on the VM, where the SSH Keys is going to be Uploaded To
	return "/ssh-pem/"
}

func (this *VirtualMachineSshManager) GetVirtualMachineUrl() (*url.URL, error) {
	// Returns Full Virtual Machine Url
	VmFolder := this.VirtualMachine.Client().ServiceContent.RootFolder.Value
	return &url.URL{Host: os.Getenv("VMWARE_SOURCE_IP"),
		User: url.UserPassword(os.Getenv("VMWARE_SOURCE_USERNAME"),
			os.Getenv("VMWARE_SOURCE_PASSWORD")),
		Path: VmFolder}, nil
}

func (this *VirtualMachineSshManager) UploadSshKeys(Key PrivateKey) error {
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

func (this *VirtualMachineSshManager) GenerateSshKeys(Manager ssh.Manager) (*PrivateKey, error) {
	// Returns Generated SSH Keys

	GeneratedCertificate := Manager.Certificate()
	PrivateKey := GeneratedCertificate.Leaf.Raw
	PublicKey := Manager.Certificate().Leaf.RawSubjectPublicKeyInfo

	var GenerationError error

	NewPrivateKeyFileError := ioutil.WriteFile("ssh-key.pem", PrivateKey, fs.FileMode(fs.ModeExclusive))
	NewPublicKeyFileError := ioutil.WriteFile("ssh-key.json", PublicKey, fs.FileMode(fs.ModeExclusive))

	if NewPrivateKeyFileError != nil || NewPublicKeyFileError != nil {
		GenerationError = errors.New("Failed to Generate SSH Keys")
	}
	return NewPrivateKey(PrivateKey, "ssh-key.pem"), GenerationError
}
