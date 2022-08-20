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
}

type Customer struct {
	gorm.Model
	Username string           `json:"Username" gorm:"type:varchar(100); not null; unique;"`
	Email    string           `json:"Email" gorm:"type:varchar(100); not null; unique;"`
	Password string           `json:"Password" gorm:"type:varchar(100); not null;"`
	Vms      []VirtualMachine `json:"Vms" gorm:"many2many:VirtualMachines;"`
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
