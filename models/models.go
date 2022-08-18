package models

import (
	"fmt"
	"log"

	"os"

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
func (this *Customer) Create() {
}

func (this *Customer) Delete() {

}

type VirtualMachine struct {
	gorm.Model

	OwnerId       string `json:"OwnerId" gorm:"type:varchar(100); not null; unique;"`
	ExternalIP    string `json:"Host" gorm:"type:varchar(100); not null; unique;"`
	ExternalPort  string `json:"Port" gorm:"type:varchar(100); not null; unique;"`
	NetworkIP     string `json:"NetworkIP" gorm:"type:varchar(100); not null;"`
	SshPublicKey  string `json:"SshPublicKey" gorm:"type:varchar(100); not null; unique;"`
	SshPrivateKey string `json:"SshPrivateKey" gorm:"type:varchar(100); not null; unique;"`
	ItemPath      string `json:"ItemPath" gorm:"type:varchar(100); not null;"`
}

func NewVirtualMachine(

	OwnerId string, // ID Of the Customer, who Owns this Virtual Machine
	ExternalIP string, // ExternalIP of the Virtual Machine
	ExternalPort string, // ExternalPort of the Virtual Machine
	NetworkIP string, // Network IP Address, the Virtual machine Is bind to
	SshPublicKey string, // Ssh Public Key to connect externally,
	SshPrivateKey string, // Ssh Private Key to validate Connections via SSH Tunnel to the Virtual Machine
	ItemPath string,
) *VirtualMachine {

	return &VirtualMachine{
		OwnerId:       OwnerId,
		ExternalIP:    ExternalIP,
		ExternalPort:  ExternalPort,
		NetworkIP:     NetworkIP,
		SshPublicKey:  SshPublicKey,
		SshPrivateKey: SshPrivateKey,
		ItemPath:      ItemPath,
	}
}
func (this *VirtualMachine) Create() {
}

func (this *VirtualMachine) Delete() {
}

type Configuration struct {
	gorm.Model

	VirtualMachineID string         `json:"VirtualMachineID" gorm:"primaryKey;unique;"`
	VirtualMachine   VirtualMachine `gorm:"foreignKey:VirtualMachine;references:VirtualMachineID;"`

	Storage      string `json:"Storage" gorm:"type:varchar(1000); not null; unique;"`
	Network      string `json:"Network" gorm:"type:varchar(1000); not null;"`
	DataCenter   string `json:"DataCenter" gorm:"type:varchar(1000); not null;"`
	DataStore    string `json:"DataStore" gorm:"type:varchar(1000); not null;"`
	ResourcePool string `json:"ResourcePool" gorm:"type:varchar(1000); not null;"`
	ItemPath     string `json:"ItemPath" gorm:"type:varchar(100); not null;"`
}

func NewConfiguration(
	SerializedStorageInfo string,
	SerializedNetworkInfo string,
	SerializedDataCenterInfo string,
	SerializedDatastoreInfo string,
	SerializedResourcePoolInfo string,
) *Configuration {

	return &Configuration{
		Storage:      SerializedStorageInfo,
		Network:      SerializedNetworkInfo,
		DataCenter:   SerializedDataCenterInfo,
		DataStore:    SerializedDatastoreInfo,
		ResourcePool: SerializedResourcePoolInfo,
	}
}
func (this *Configuration) Create() {
}

func (this *Configuration) Delete() {
}
