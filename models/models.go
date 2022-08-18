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
	DATABASE_NAME     = os.Getenv("DATABASE_NAME")
	DATABASE_HOST     = os.Getenv("DATABASE_HOST")
	DATABASE_PORT     = os.Getenv("DATABASE_PORT")
	DATABASE_USER     = os.Getenv("DATABASE_USER")
	DATABASE_PASSWORD = os.Getenv("DATABASE_PASSWORD")
)

func init() {
	Database, ConnectionError := gorm.Open(postgres.New(postgres.Config{
		DSN: fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
			DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME),
	}))
	if ConnectionError != nil {
		panic(ConnectionError)
	}
	Database.AutoMigrate(&Customer{}, &VirtualMachine{}, &Configuration{})
}

type Customer struct {
	gorm.Model

	Username string           `json:"Username" gorm:"type:varchar(100); not null; unique;"`
	Email    string           `json:"Email" gorm:"type:varchar(100); not null; unique;"`
	Password string           `json:"Password" gorm:"type:varchar(100); not null;"`
	Vms      []VirtualMachine `json:"Vms" gorm:"many2many:VirtualMachine;"`
}

type VirtualMachine struct {
	gorm.Model

	OwnerId       string `json:"OwnerId" gorm:"type:varchar(100); not null; unique;"`
	ExternalIP    string `json:"Host" gorm:"type:varchar(100); not null; unique;"`
	ExternalPort  string `json:"Port" gorm:"type:varchar(100); not null; unique;"`
	NetworkIP     string `json:"NetworkIP" gorm:"type:varchar(100); not null;"`
	SshPublicKey  string `json:"SshPublicKey" gorm:"type:varchar(100); not null; unique;"`
	SshPrivateKey string `json:"SshPrivateKey" gorm:"type:varchar(100); not null; unique;"`
}

type Configuration struct {
	gorm.Model

	VirtualMachineID string         `json:"VirtualMachineID"`
	VirtualMachine   VirtualMachine `gorm:"foreignKey:VirtualMachine;references:"`

	Storage      string `json:"Storage" gorm:"type:varchar(1000); not null; unique;"`
	Network      string `json:"Network" gorm:"type:varchar(1000); not null;"`
	DataCenter   string `json:"DataCenter" gorm:"type:varchar(1000); not null;"`
	DataStore    string `json:"DataStore" gorm:"type:varchar(1000); not null;"`
	ResourcePool string `json:"ResourcePool" gorm:"type:varchar(1000); not null;"`
}
