package exceptions

import (
	"errors"
	"fmt"
)

func IPSetupFailure() error {
	return errors.New("Failed to Setup IP Address")
}

func NetworkSetupFailure() error {
	return errors.New("Failed to Setup Network")
}

func ResourcesSetupFailure() error {
	return errors.New("Failed to Setup Resources, to the Virtual Machine")
}

func StorageSetupFailure() error {
	return errors.New("Failed to setup Storage")
}

func VMDeployFailure() error {
	return errors.New("Failed to Deploy Virtual Machine")
}

func VMShutdownFailure() error {
	return errors.New("Failed to Shutdown Virtual Machine")
}

func DeployFromLibraryFailure() error {
	return errors.New("Failed to Deploy Virtual Machine from the Library")
}

func NoResourceAvailable() error {
	return errors.New("No That Amount of Resource available Now :(")
}

func ItemDoesNotExist() error {
	return errors.New("Resource Item does not Exist.")
}

func DestroyFailure() error {
	return errors.New("Failed to Destroy Virtual Machine")
}

func ComponentDoesNotExist(ComponentName string) error {
	return errors.New(fmt.Sprintf("Resource: %s does not Exist", ComponentName))
}
