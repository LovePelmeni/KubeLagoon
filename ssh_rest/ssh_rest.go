package ssh_rest

import (
	"net/http"

	"log"
	"os"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("SshRest.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

func GetCustomerVirtualMachineSSHKeysRestController(RequestContext *gin.Context) {

	var Query []struct {
		gorm.Model
		VirtualMachine models.VirtualMachine // Vm Server
		SSHPublicKey   models.SSHPublicKey   // SSHPublicKey to Access this Server
	} // Represents query of the Joins Pattern

	CustomerId := RequestContext.Query("CustomerId")
	QuerySet := models.Database.Model(&models.VirtualMachine{}).Where("id = ?", CustomerId).Joins(
		"JOIN ssh_public_key ON virtual_machine.id = ssh_public_key.virtual_machine_id").Find(&Query)

	if QuerySet.Error != nil {
		ErrorLogger.Printf(
			"Failed to Obtain the QuerySet of Customer SSH Public Keys, Error: %s", QuerySet.Error)
		RequestContext.JSON(
			http.StatusBadRequest, gin.H{"Error": "Failed to Get Vm SSH Keys"})
	}
	RequestContext.JSON(http.StatusOK, gin.H{"QuerySet": Query})
}

func UpdateVirtualMachineSshKeysRestController(RequestContext *gin.Context) {
	// Rest Controller, that Allows to Update SSH Key Pairs with new Ones
}
