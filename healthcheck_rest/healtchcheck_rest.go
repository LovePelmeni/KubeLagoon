package healthcheck_rest

import "github.com/gin-gonic/gin"

// package provides Rest API Controllers, that Provides Info about the Virtual Machine Server Health Metrics

func GetVirtualMachineCPUInfoRestController(RequestContext *gin.Context) {
	// Rest Controller, that Provides Info about CPU State of the Virtual Machine
}

func GetVirtualMachineMemoryInfoRestController(RequestContext *gin.Context) {
	// Rest Controller, that Provides Info about the Memory State of the Virtual Machine
}
