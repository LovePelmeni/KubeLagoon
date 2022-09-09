package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	Logger *zap.Logger
)

var (
	Database *gorm.DB
)

const StatusNotReady = "NotReady" // Defines the Status of the Virtual Machine Availability
const StatusReady = "Ready"       // Defines the Status of Virtual Machine Availability

var (
	DATABASE_NAME     = os.Getenv("DATABASE_NAME")
	DATABASE_HOST     = os.Getenv("DATABASE_HOST")
	DATABASE_PORT     = os.Getenv("DATABASE_PORT")
	DATABASE_USER     = os.Getenv("DATABASE_USER")
	DATABASE_PASSWORD = os.Getenv("DATABASE_PASSWORD")
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
	DatabaseInstance, ConnectionError := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
			DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME),
	}))

	switch ConnectionError {

	case gorm.ErrInvalidDB:
		panic("Please Setup Credentials for your PostgreSQL Database: Host, Port, User, Password, DbName")

	case gorm.ErrUnsupportedDriver:
		panic("Invalid Database Driver")

	case gorm.ErrNotImplemented:
		panic(ConnectionError)
	}

	Database = DatabaseInstance
	Database.AutoMigrate(&Customer{}, &VirtualMachine{})
	InitializeProductionLogger()
}

type Customer struct {
	// Customer Database ORM Model
	ID       int
	Username string `json:"Username" gorm:"<-:create;type:varchar(100); not null; unique;"`
	Email    string `json:"Email" gorm:"<-:create;type:varchar(100); not null; unique;"`
	Password string `json:"Password" gorm:"type:varchar(100); not null;"`

	BillAddress string `json:"BillAddress" xml:"BillAddress" gorm:"type:varchar(100); not null;"`
	Country     string `json:"Country" xml:"Country" gorm:"type:varchar(100); not null;"`
	ZipCode     string `json:"ZipCode" xml:"ZipCode" gorm:"type:varchar(100); not null;"`
}

func NewCustomer(Username string, Password string, Email string) *Customer {
	PasswordHash, HashError := bcrypt.GenerateFromPassword([]byte(Password), 14)
	if HashError != nil {
		return nil
	}
	return &Customer{
		Username: Username,
		Email:    Email,
		Password: string(PasswordHash),
	}
}

func (this *Customer) Create() (*gorm.DB, error) {
	// Creates New Customer Profile
	CreatedCustomer := Database.Model(&Customer{}).Create(this)
	return CreatedCustomer, CreatedCustomer.Error
}

func (this *Customer) Delete(UserId int) (*gorm.DB, error) {
	// Deletes Customer Profile
	DeletedCustomer := Database.Where("id = ?", UserId).Delete(&Customer{})
	Database.Unscoped().Where("id = ?", UserId).Delete(&Customer{})
	return DeletedCustomer, DeletedCustomer.Error
}

// NOTE: Going to support SSL soon

type LoadBalancerService struct {
	// Traefik Service, that is used to proxy requests to the Virtual Machine
	ID               int
	IPAddress        string `json:"IPAddress" gorm:"type:varchar(100); not null; unique;"`
	VirtualMachineId int    `json:"VirtualMachineId" gorm:"type:integer"`
}

func NewLoadBalancer(IPAddress string, VirtualMachineId int) *LoadBalancerService {
	// Returns New Database instance of the Load Balancer
	return &LoadBalancerService{
		IPAddress:        IPAddress,
		VirtualMachineId: VirtualMachineId,
	}
}

func (this *LoadBalancerService) Create() (*gorm.DB, error) {
	// Creates new Instance of the Load Balancers
	Created := Database.Create(&this)
	return Created, Created.Error
}

func (this *LoadBalancerService) Delete() (*gorm.DB, error) {
	Deleted := Database.Delete(&this)
	return Deleted, Deleted.Error
}

type VirtualMachine struct {
	ID                 int
	State              string                      `json:"State" xml:"State" gorm:"type:varchar(10); not null;"`
	SshInfo            SSHInfo                     `json:"sshKey" xml:"sshKey" gorm:"column:ssh_key;type:text;default:null;"`
	Configuration      VirtualMachineConfiguration `json:"Configuration" xml:"Configuration" gorm:"column:configuration;type:text;default:null;"`
	OwnerId            int                         `json:"OwnerId" xml:"OwnerId" gorm:"<-:create;type:varchar(100);not null;unique;"`
	VirtualMachineName string                      `json:"VirtualMachineName" xml:"VirtualMachineName" gorm:"type:varchar(100);not null;"`
	ItemPath           string                      `json:"ItemPath" xml:"ItemPath" gorm:"<-:create;type:varchar(100);not null;"`
	IPAddress          string                      `json:"IPAddress" xml:"IPAddress" gorm:"<-:create;type:varchar(100);not null;unique;"`
}

func NewVirtualMachine(

	OwnerId int, // ID Of the Customer, who Owns this Virtual Machine
	VirtualMachineName string, // Virtual Machine UniqueName
	SshInfo *SSHInfo, // SSH Info, defines what method and credentials to use, In Order to Connect to the VM Server
	ItemPath string,
	IPAddress string,
	Configuration ...*VirtualMachineConfiguration,

) *VirtualMachine {

	return &VirtualMachine{
		OwnerId:            OwnerId,
		VirtualMachineName: VirtualMachineName,
		ItemPath:           ItemPath,
		IPAddress:          IPAddress,
		Configuration:      *Configuration[0],
		SshInfo:            *SshInfo,
	}
}

func (this *VirtualMachine) Save() (*gorm.DB, error) {
	// Saved the Current Virtual Machine Object
	Saved := Database.Save(this)
	return Saved, Saved.Error
}

func (this *VirtualMachine) Create() (*gorm.DB, error) {
	// Creates New Virtual Machine Object

	Created := Database.Clauses(clause.OnConflict{Columns: []clause.Column{
		{Table: "VirtualMachine", Name: "VirtualMachineName"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"VirtualMachineName": gorm.Expr("virtual_machine_name + _ + uuid_generate_v3()"),
		})}).Create(&this)
	return Created, Created.Error
}

func (this *VirtualMachine) Delete() (*gorm.DB, error) {
	// Deletes the Virtual Machine ORM Object....

	Deleted := Database.Clauses(clause.OnConflict{DoNothing: true}).Delete(&this)
	Database.Model(&VirtualMachine{}).Unscoped().Delete(&this)
	return Deleted, Deleted.Error
}

type VirtualMachineConfiguration struct {
	// Virtual Machine Configuration

	// Metadata about the Virtual Machine

	Metadata struct {
		VirtualMachineId string `json:"VirtualMachineId" xml:"VirtualMachineId"`
		VmOwnerId        string `json:"VmOwnerId" xml:"VmOwnerId"`
	} `json:"Metadata" xml:"Metadata"`

	// Load Balancer Configuration

	LoadBalancer struct {

		// Proxies Traffic From...

		LoadBalancerServiceName string `json:"LoadBalancerName" xml:"LoadBalancerName"`
		LoadBalancerHost        string `json:"LoadBalancerHost,omitempty;" xml:"LoadBalancerHost"`
		LoadBalancerPort        string `json:"LoadBalancerPort" xml:"LoadBalancerPort"`

		// Proxies Traffic to...
		VirtualMachineHost string `json:"ProxyHost" xml:"ProxyHost"`
		VirtualMachinePort string `json:"ProxyPort" xml:"ProxyPort"`
	} `json:"LoadBalancer" xml:"LoadBalancer"`

	// Host System Configuration

	HostSystem struct {
		Type             string `json:"Type"` // OS Distribution Type Like: Linux, Windows etc....
		DistributionName string `json:"DistributionName"`
		Bit              int64  `json:"Bit;omitempty"`
	} `json:"HostSystem" xml:"HostSystem"`

	// Internal Network Configuration

	Network struct {
		IP       string `json:"IP,omitempty"`
		Netmask  string `json:"Netmask,omitempty"`
		Hostname string `json:"Hostname,omitempty"`
		Gateway  string `json:"Gateway,omitempty"`
		Enablev6 bool   `json:"Enablev6,omitempty"`
		Enablev4 bool   `json:"Enablev4,omitempty"`
	} `json:"Network" xml:"Network"`

	// Extra Tools, that is going to be Installed on the VM automatically
	// Things Like Docker, Docker-Compose, VirtualBox or Podman etc....

	ExtraTools struct {
		Tools []string `json:"Tools" xml:"Tools"` // Names of the Tools
	} `json:"ExtraTools;omitempty" xml:"ExtraTools"`

	// Hardware Resourcs for the VM Configuration

	Resources struct {
		CpuNum            int32 `json:"CpuNum" xml:"CpuNum"`
		MemoryInMegabytes int64 `json:"MemoryInMegabytes" xml:"MemoryInMegabytes"`
		MaxMemoryUsage    int64 `json:"MaxMemoryUsage,omitempty;" xml:"MaxMemoryUsage"`
		MaxCpuUsage       int64 `json:"MaxCpuUsage,omitempty;" xml:"MaxCpuUsage"`
	} `json:"Resources" xml:"Resources"`

	Ssh struct {
		Type string `json:"Type" xml:"Type"`
	} `json:"Ssh" xml:"Ssh"`

	Disk struct {
		CapacityInKB int `json:"CapacityInKB" xml:"CapacityInKB"`
	} `json:"Disk"`
}

func NewVirtualMachineConfiguration(SerializedConfiguration []byte) (*VirtualMachineConfiguration, error) {
	// Returns New Serialized Virtual Machine Configuration
	var NewConfiguration VirtualMachineConfiguration
	DecodedConfigurationError := json.Unmarshal(SerializedConfiguration, &NewConfiguration)
	return &NewConfiguration, DecodedConfigurationError
}

// Sql Methods for managing Encoding and Decoding of the SQL Model

func (this *VirtualMachineConfiguration) Scan(source interface{}) error {
	return json.Unmarshal(source.([]byte), &this)
}

func (this *VirtualMachineConfiguration) Value() (driver.Value, error) {
	EncodedData, Error := json.Marshal(this)
	return string(EncodedData), Error
}

const TypeByRootCredentials = "ByRootCredentials"

const TypeByRootCertificate = "ByRootCertificate"

type SSHInfo struct {
	// Depending on the Type of the SSH Info, it can be via SSL Certificate or via Root Credentials
	// So the Info Going to be Serialzied into json and put inside the `SshCredentialsInfo` Field
	Type               string `json:"Type" xml:"Type"`
	SshCredentialsInfo string `json:"SshCredentialsInfo" xml:"SshCredentialsInfo"`
	VirtualMachineId   int    `json:"VirtualMachineId" xml:"VirtualMachineId"`
}

func NewSshPublicKey(Type string, SshInfo []byte, VirtualMachineId int) *SSHInfo {
	return &SSHInfo{
		Type:               Type,
		SshCredentialsInfo: string(SshInfo),
		VirtualMachineId:   VirtualMachineId,
	}
}
func (this *SSHInfo) Scan(inter interface{}) error {
	return json.Unmarshal(inter.([]byte), this)
}

func (this *SSHInfo) Value() ([]byte, error) {
	Serialized, Error := json.Marshal(this)
	return Serialized, Error
}
