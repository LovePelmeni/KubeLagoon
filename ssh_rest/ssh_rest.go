package ssh_rest

import (
	"net/http"

	"os"
	"strconv"

	"github.com/LovePelmeni/Infrastructure/authentication"
	"github.com/LovePelmeni/Infrastructure/deploy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/LovePelmeni/Infrastructure/models"
	"github.com/LovePelmeni/Infrastructure/ssh_config"

	"github.com/gin-gonic/gin"
	"github.com/vmware/govmomi/vim25"
)

var (
	Logger *zap.Logger
)

func InitializeProductionLogger() {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(config)
	file, _ := os.OpenFile("ResourcesLog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	logWriter := zapcore.AddSync(file)

	Core := zapcore.NewTee(zapcore.NewCore(fileEncoder, logWriter, zapcore.DebugLevel))
	Logger = zap.New(Core)
}

func init() {
	InitializeProductionLogger()
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
			Logger.Error("Failed to Update SSH keys for the VM wit ID: %s",
				zap.String("Virtual Machine ID", VirtualMachineId),
				zap.String("Virtual Machine Owner", jwtCredentials.Id))
		}
	default:
		RequestContext.JSON(http.StatusBadGateway,
			gin.H{"Error": "Failed to Generate New SSH Keys"})
	}
}
