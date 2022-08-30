package ssh_rest

import (
	"log"
	"net/http"

	"os"
	"strconv"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/deploy"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/ssh_config"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi/vim25"
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

func GetCustomerVirtualMachineSSHCredentialsRestController(RequestContext *gin.Context) {
	// Rest Controller, that is responsible for giving SSH Info of Customer's Virtual Machines

	var Query []struct {
		gorm.Model
		VirtualMachine models.VirtualMachine // Vm Server
		SSHInfo        models.SSHInfo        // SSHPublicKey to Access this Server
	} // Represents query of the Joins Pattern

	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(
		RequestContext.Request.Header.Get("Authorization"))
	CustomerId := jwtCredentials.UserId

	var Queryset []models.VirtualMachine
	models.Database.Model(
		&models.VirtualMachine{}).Where("id = ?", CustomerId).Find(&Query)

	for _, ModelObj := range Queryset {
		StructedData := struct {
			gorm.Model
			VirtualMachine models.VirtualMachine // Vm Server
			SSHInfo        models.SSHInfo        // SSHPublicKey to Access this Server
		}{
			VirtualMachine: ModelObj,
			SSHInfo:        ModelObj.SshInfo,
		}
		Query = append(Query, StructedData)
	}
	RequestContext.JSON(http.StatusOK,
		gin.H{"QuerySet": Query})
}

func RecoverSSHKeyRestController(RequestContext *gin.Context) {
	// Recovering SSH Keys, by picking them out from the Temp Buffer
}

func UpdateVirtualMachineSshKeysRestController(RequestContext *gin.Context) {
	// Rest Controller, that Allows to Update SSH Key Pairs with new Ones

	jwtCredentials, _ := authentication.GetCustomerJwtCredentials(
		RequestContext.Request.Header.Get("Authentication"))

	VirtualMachineId := RequestContext.Query("VirtualMachineId")
	VmOwnerId := jwtCredentials.UserId

	var VirtualMachineModel models.VirtualMachine
	models.Database.Model(&models.VirtualMachine{}).Where("id = ?", VirtualMachineId).Find(&VirtualMachineModel)
	VmManager := deploy.NewVirtualMachineManager(vim25.Client{})
	VirtualMachine, FindError := VmManager.GetVirtualMachine(VirtualMachineId, strconv.Itoa(VmOwnerId))

	if FindError != nil {
		RequestContext.JSON(http.StatusBadRequest,
			gin.H{"Error": "Virtual Machine Server not Found"})
		return
	}

	SshManager := ssh_config.NewVirtualMachineSshCertificateManager(vim25.Client{}, VirtualMachine)
	PublicKey, GenerateError := SshManager.GenerateSshKeys()

	if GenerateError != nil {
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Generate New SSH Keys"})
		return
	}

	switch GenerateError {
	case nil:

		Gorm := models.Database.Model(
			&models.VirtualMachine{}).Where(
			"id = ? AND owner_id = ?").Unscoped().Update("ssh_public_key", PublicKey)

		if Gorm.Error != nil {
			Gorm.Rollback()
			ErrorLogger.Printf("Failed to Update SSH keys for the VM wit ID: %s", VirtualMachineId)
		}
	default:
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Generate New SSH Keys"})
	}
}
