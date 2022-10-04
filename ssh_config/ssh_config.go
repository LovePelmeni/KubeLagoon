package ssh_config

import (
	"context"
	"errors"

	"fmt"

	"os"
	"time"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/google/uuid"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"golang.org/x/crypto/bcrypt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("Main.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
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
	// SSH Certificate Credentials for the Virtual Server
	FileName string `json:"FileName" xml:"FileName"`
	Content  []byte `json:"Content" xml:"Content"`
}

func NewSshCertificateCredentials(Content []byte, FileName string) *SshCertificateCredentials {
	return &SshCertificateCredentials{
		FileName: FileName,
		Content:  Content,
	}
}

type VirtualMachineSshManagerInterface interface {
	// Interface, represents base SSH Manager Interface for the Virtual Machine Server
}

type VirtualMachineSshCertificateManager struct {
	VirtualMachineSshManagerInterface
	Client vim25.Client
}

func NewVirtualMachineSshCertificateManager(Client vim25.Client) *VirtualMachineSshCertificateManager {
	return &VirtualMachineSshCertificateManager{
		Client: Client,
	}
}

func (this *VirtualMachineSshCertificateManager) UploadSshKeys(VirtualMachine *object.VirtualMachine, Key SshCertificateCredentials) error {
	// Uploaded SSH Pem Key to the Virtual Machine Server...

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	// Initializing New SSH Certificate Manager
	HostSystem, HostSystemError := VirtualMachine.HostSystem(TimeoutContext)
	if HostSystemError != nil {
		return errors.New("Failed to Get Host System Info")
	}
	SshManager := object.NewHostCertificateManager(&this.Client,
		*types.NewReference(this.Client.ServiceContent.RootFolder), HostSystem.Reference())

	// Uploading SSL Certificate to the Host Machine
	InstallationError := SshManager.InstallServerCertificate(TimeoutContext, string(Key.Content))
	switch InstallationError {
	case nil:
		Logger.Debug("SSH Key has been Successfully Uploaded to the VM with Name: %s",
			zap.String("Virtual Machine Name", VirtualMachine.Name()))
		return nil

	default:
		Logger.Error("Failed to Upload SSH Key to the Remote VM's Host Machine")
		return errors.New("Failed to Add SSH Support")
	}
}

func (this *VirtualMachineSshCertificateManager) GenerateSshKeys(VirtualMachine *object.VirtualMachine, VirtualMachineId string) (*SshCertificateCredentials, error) {

	// Returns Generated SSH Keys for the Virtual Machine Server

	// Certificate will be Generated with the Specific Name and will be Stored on the Host System
	// Of the Virtual Machine Server

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	HostSystem, HostSystemError := VirtualMachine.HostSystem(TimeoutContext)
	if HostSystemError != nil {
		return nil, errors.New("Failed to Get OS Info")
	}
	// Initializing Manager for the SSH Management
	Manager := object.NewHostCertificateManager(
		&this.Client, *types.NewReference(this.Client.ServiceContent.RootFolder), HostSystem.Reference())

	SSLCertificateDistinguishName := fmt.Sprintf("%s-%s", VirtualMachine.Name(), VirtualMachineId)
	Manager.Client().Certificate().Leaf.MaxPathLen = 30 // Initializing Max Path len for the Virtual Machine Server
	GeneratedCertificate, GenerationError := Manager.GenerateCertificateSigningRequestByDn(TimeoutContext, SSLCertificateDistinguishName)

	// Returning the Response
	currentTime := time.Now()
	return NewSshCertificateCredentials(
		[]byte(GeneratedCertificate),
		fmt.Sprintf("%s.%s.pub", VirtualMachineId, currentTime),
	), GenerationError
}

type VirtualMachineSshRootCredentialsManager struct {
	// SSH Manager Class, that performs Type of the SSH Connection
	// Via Root Credentials
	VirtualMachineSshManagerInterface
	Client vim25.Client
}

func NewVirtualMachineSshRootCredentialsManager(Client vim25.Client) *VirtualMachineSshRootCredentialsManager {
	return &VirtualMachineSshRootCredentialsManager{
		Client: Client,
	}
}

func (this *VirtualMachineSshCertificateManager) GetSshRootUserCredentials(VirtualMachineId string) models.SSHConfiguration {

	// Returns Info about the Ssh Root Credentials of the Virtual Machine Server
	// Is working only with the Vm's which has the `Root User Credentials` Type

	var VirtualMachine models.VirtualMachine
	models.Database.Model(&models.VirtualMachine{}).Where(
		"id = ?", VirtualMachineId).Find(&VirtualMachine)
	return VirtualMachine.SshInfo
}

func (this *VirtualMachineSshRootCredentialsManager) GetSshRootCredentials(VirtualMachine *object.VirtualMachine) (*types.NamePasswordAuthentication, error) {
	// Parses Root Credentials of the OS Host System of the Customer's Virtual Machine Server
	// The Returned object `types.GuestAuthentication` can be potentially used for making operations
	// that requires this authentication

	PasswordUuid := uuid.New()
	GeneratedOsPassword, _ := bcrypt.GenerateFromPassword([]byte(PasswordUuid.String()), 15)
	SshCredentials := types.NamePasswordAuthentication{
		Username: "root",
		Password: string(GeneratedOsPassword),
	}
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	Manager := property.DefaultCollector(&this.Client)
	defer CancelFunc()

	// Receiving Virtual Machine Instance

	RetrieveError := Manager.RetrieveOne(TimeoutContext, VirtualMachine.Reference(),
		[]string{"name", "guest"}, &VirtualMachine)

	if RetrieveError != nil {
		Logger.Debug(
			"Failed to Get VirtualMachine Instance")
		return nil, RetrieveError
	}
	return &SshCredentials, nil
}
