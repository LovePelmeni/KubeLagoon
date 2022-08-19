package models

import (
	"encoding/json"
	"fmt"
	"log"

	"os"

	"github.com/LovePelmeni/Infrastructure/parsers"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	if ConnectionError != nil {
		panic(ConnectionError)
	}
	Database = DatabaseInstance
	Database.AutoMigrate(&Customer{}, &VirtualMachine{}, &Configuration{})
}

type Customer struct {
	gorm.Model
	Username string           `json:"Username" gorm:"type:varchar(100); not null; unique;"`
	Email    string           `json:"Email" gorm:"type:varchar(100); not null; unique;"`
	Password string           `json:"Password" gorm:"type:varchar(100); not null;"`
	Vms      []VirtualMachine `json:"Vms" gorm:"many2many:VirtualMachine;"`
}

func NewCustomer() *Customer {
	return &Customer{}
}
func (this *Customer) Create() (*gorm.DB, error) {
	CreatedCustomer := Database.Model(&Customer{}).Create(&this)
	return CreatedCustomer, CreatedCustomer.Error
}

func (this *Customer) Delete() (*gorm.DB, error) {
	DeletedCustomer := Database.Model(&Customer{}).Delete(&this)
	Database.Model(&Customer{}).Unscoped().Delete(&this)
	return DeletedCustomer, DeletedCustomer.Error
}

type VirtualMachine struct {
	gorm.Model

	OwnerId            string `json:"OwnerId" gorm:"type:varchar(100); not null; unique;"`
	ExternalIP         string `json:"Host" gorm:"type:varchar(100); not null; unique;"`
	ExternalPort       string `json:"Port" gorm:"type:varchar(100); not null; unique;"`
	VirtualMachineName string `json:"VirtualMachineName" gorm:"type:varchar(100); not null;"`
	ItemPath           string `json:"ItemPath" gorm:"type:varchar(100); not null;"`
}

func NewVirtualMachine(

	OwnerId string, // ID Of the Customer, who Owns this Virtual Machine
	VirtualMachineName string, // Virtual Machine UniqueName
	ItemPath string,

) *VirtualMachine {

	return &VirtualMachine{
		OwnerId:            OwnerId,
		VirtualMachineName: VirtualMachineName,
		ItemPath:           ItemPath,
	}
}

func (this *VirtualMachine) Create() (*gorm.DB, error) {

	Created := Database.Model(&VirtualMachine{}).Create(&this)
	return Created, Created.Error
}

func (this *VirtualMachine) Delete() (*gorm.DB, error) {
	Deleted := Database.Model(&VirtualMachine{}).Delete(&this)
	Database.Model(&VirtualMachine{}).Unscoped().Delete(&this)
	return Deleted, Deleted.Error
}

type Configuration struct {
	gorm.Model

	VirtualMachineID string         `json:"VirtualMachineID" gorm:"primaryKey;unique;"`
	VirtualMachine   VirtualMachine `gorm:"foreignKey:VirtualMachine;references:VirtualMachineID;"`

	Disk         string `json:"Storage" gorm:"type:varchar(1000); not null; unique;"`
	Network      string `json:"Network" gorm:"type:varchar(1000); not null;"`
	DataCenter   string `json:"DataCenter" gorm:"type:varchar(1000); not null;"`
	DataStore    string `json:"DataStore" gorm:"type:varchar(1000); not null;"`
	ResourcePool string `json:"ResourcePool" gorm:"type:varchar(1000); not null;"`
	ItemPath     string `json:"ItemPath" gorm:"type:varchar(100); not null;"`
	Folder       string `json:"Folder" xml:"Folder" gorm:"type:varchar(1000); not null;"`
}

func NewConfiguration(
	Config parsers.HardwareConfig,
	CustomConfig parsers.VirtualMachineCustomSpec,
) *Configuration {

	SerializedDatacenterConfig, _ := json.Marshal(Config.Datacenter)
	SerializedDatastoreConfig, _ := json.Marshal(Config.DataStore)
	SerializedNetworkConfig, _ := json.Marshal(Config.Network)
	SerializedResourcePoolConfig, _ := json.Marshal(CustomConfig.Resources)
	SerializedFolderConfig, _ := json.Marshal(Config.Folder)
	SerializedDiskConfig, _ := json.Marshal(CustomConfig.Disk)

	return &Configuration{
		Disk:         string(SerializedDiskConfig),
		Network:      string(SerializedNetworkConfig),
		DataCenter:   string(SerializedDatacenterConfig),
		DataStore:    string(SerializedDatastoreConfig),
		ResourcePool: string(SerializedResourcePoolConfig),
		Folder:       string(SerializedFolderConfig),
	}
}
func (this *Configuration) Create() (*gorm.DB, error) {
	Created := Database.Model(&Configuration{}).Create(&this)
	return Created, Created.Error
}

func (this *Configuration) Delete() (*gorm.DB, error) {
	Deleted := Database.Model(&Configuration{}).Delete(&this)
	Database.Model(&Configuration{}).Unscoped().Delete(&this)
	return Deleted, Deleted.Error
}
