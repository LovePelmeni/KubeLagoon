package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

var (
	Database *gorm.DB
)

var (
	DATABASE_NAME     = os.Getenv("DATABASE_NAME")
	DATABASE_HOST     = os.Getenv("DATABASE_HOST")
	DATABASE_PORT     = os.Getenv("DATABASE_PORT")
	DATABASE_USER     = os.Getenv("DATABASE_USER")
	DATABASE_PASSWORD = os.Getenv("DATABASE_PASSWORD")
)

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
	LogFile, Error := os.OpenFile("Models.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

type Customer struct {
	ID       int
	Username string `json:"Username" gorm:"<-:create;type:varchar(100); not null; unique;"`
	Email    string `json:"Email" gorm:"<-:create;type:varchar(100); not null; unique;"`
	Password string `json:"Password" gorm:"type:varchar(100); not null;"`
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

type VirtualMachine struct {
	ID                 int
	SshKey             SSHPublicKey                `json:"sshKey" xml:"sshKey" gorm:"column:ssh_key;type:text;"`
	Configuration      VirtualMachineConfiguration `json:"Configuration" xml:"Configuration" gorm:"column:configuration;type:text;"`
	OwnerId            string                      `json:"OwnerId" xml:"OwnerId" gorm:"<-:create;type:varchar(100);not null;unique;"`
	VirtualMachineName string                      `json:"VirtualMachineName" xml:"VirtualMachineName" gorm:"type:varchar(100);not null;"`
	ItemPath           string                      `json:"ItemPath" xml:"ItemPath" gorm:"<-:create;type:varchar(100);not null;"`
	IPAddress          string                      `json:"IPAddress" xml:"IPAddress" gorm:"<-:create;type:varchar(100);not null;unique;"`
}

func NewVirtualMachine(

	OwnerId string, // ID Of the Customer, who Owns this Virtual Machine
	VirtualMachineName string, // Virtual Machine UniqueName
	SshKey SSHPublicKey,
	Configuration VirtualMachineConfiguration,
	ItemPath string,
	IPAddress string,

) *VirtualMachine {

	return &VirtualMachine{
		OwnerId:            OwnerId,
		VirtualMachineName: VirtualMachineName,
		ItemPath:           ItemPath,
		IPAddress:          IPAddress,
		Configuration:      Configuration,
		SshKey:             SshKey,
	}
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

type SSHPublicKey struct {
	FilePath         string `json:"FilePath" xml:"FilePath"`
	Key              string `json:"Key" xml:"Key"`
	Filename         string `json:"Filename" xml:"Filename"`
	VirtualMachineId int    `json:"VirtualMachineId" xml:"VirtualMachineId"`
}

func NewSshPublicKey(KeyContent []byte, FilePath string, Filename string) *SSHPublicKey {
	return &SSHPublicKey{
		FilePath: FilePath,
		Key:      string(KeyContent),
		Filename: Filename,
	}
}
func (this *SSHPublicKey) Scan(inter interface{}) error {
	return json.Unmarshal(inter.([]byte), this)
}

func (this *SSHPublicKey) Value() ([]byte, error) {
	Serialized, Error := json.Marshal(this)
	return Serialized, Error
}

type VirtualMachineConfiguration struct {
	// Virtual Machine Configuration
	Metadata struct {
		VirtualMachineId string `json:"VirtualMachineId" xml:"VirtualMachineId"`
		VmOwnerId        string `json:"VmOwnerId" xml:"VmOwnerId"`
	} `json:"Metadata" xml:"Metadata"`

	HostSystem struct {
		Type             string `json:"Type"` // OS Distribution Type Like: Linux, Windows etc....
		DistributionName string `json:"DistributionName"`
		Bit              int64  `json:"Bit;omitempty"`
	} `json:"HostSystem" xml:"HostSystem"`

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
