package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	_ "reflect"
	"strings"
	"time"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"

	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	DebugLogger *log.Logger
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	LogFile, Error := os.OpenFile("Resources.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG:", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

// Package consists of API Classes, that is responsible for Providing info about
// Available Hardware Instances

type ResourceRequirements interface {
	// Interface, represents Requirements (Filter), that is used to Filter the Resources
}

type DatacenterResourceRequirements struct {
	ResourceRequirements

	// Resource Requirements
	HostSystemResourceRequirements HostSystemResourceRequirements     `json:"HostSystemRequirements;"`
	NetworkResourceRequirements    NetworkResourceRequirements        `json:"NetworkRequirements;omitempty;"`
	DatastoreResourceRequirements  DatastoreResourceRequirements      `json:"DatastoreRequirements;omitempty;"`
	StorageResourceRequirements    StorageResourceRequirements        `json:"StorageRequirements;omitempty;"`
	ClusterResourceRequirements    ClusterComputeResourceRequirements `json:"ClusterComputeRequirements;omitempty"`
	FolderResourceRequirements     FolderResourceRequirements         `json:"FolderRequirements;omitempty"`
}

func NewDatacenterResourceRequirements(Requirements string) (*DatacenterResourceRequirements, error) {
	var DecodedRequirements *DatacenterResourceRequirements
	DecodeError := json.Unmarshal([]byte(Requirements), &DecodedRequirements)
	if DecodeError != nil {
		return nil, DecodeError
	}
	return DecodedRequirements, nil
}

type NetworkResourceRequirements struct {
	ResourceRequirements
	Private struct {
		IpSubnetAddr string `json:"IpSubnetAddr; omitempty;"`
		IpSubnetMask string `json:"IpSubnetMask; omitempty;"`
	} `json:"Private;omitempty"`
}

func NewNetworkResourceRequirements() *NetworkResourceRequirements {
	return &NetworkResourceRequirements{}
}

type HostSystemResourceRequirements struct {
	ResourceRequirements
	SystemName string `json:"SystemName"`
	Bit        int64  `json:"Bit"`
}

func NewHostSystemResourceRequirements(SystemName string, Bit int64) *HostSystemResourceRequirements {
	return &HostSystemResourceRequirements{
		SystemName: strings.ToLower(SystemName),
		Bit:        Bit,
	}
}

type DatastoreResourceRequirements struct {
	ResourceRequirements
	FreeSpace int32
	Capacity  int64
}

func NewDatastoreResourceRequirements(FreeSpace int32, Capacity int64) *DatastoreResourceRequirements {
	return &DatastoreResourceRequirements{}
}

type StorageResourceRequirements struct {
	ResourceRequirements
	FreeSpace int32 `json:"FreeSpace"`
	Capacity  int64 `json:"Capacity"`
}

func NewStorageResourceRequirements() *StorageResourceRequirements {
	return &StorageResourceRequirements{}
}

type ClusterComputeResourceRequirements struct {
	ResourceRequirements
	CpuNum            int32 `json:"CpuNum"`
	MemoryInMegabytes int64 `json:"MemoryInMegabytes"`
}

func NewClusterComputeRequirements() *ClusterComputeResourceRequirements {
	return &ClusterComputeResourceRequirements{}
}

type FolderResourceRequirements struct {
	ResourceRequirements
}

func NewFolderResourceRequirements() *FolderResourceRequirements {
	return &FolderResourceRequirements{}
}

type ResourceManagerInterface interface {
	// Interface, Reprensents Resource Manager, that Is Going to be
	// Providing Info about the Specific Resource
	HasEnoughResources(Datacenter *mo.Datacenter, Requirements ResourceRequirements) bool
	GetAvailableResources(Datacenter *mo.Datacenter, Requirements ResourceRequirements) ([]*types.ManagedObjectReference, error)
}

type DatacenterResourceManager struct {
	Client *vim25.Client
}

func NewDatacenterResourceManager(Client *vim25.Client) *DatacenterResourceManager {
	return &DatacenterResourceManager{
		Client: Client,
	}
}

func (this *DatacenterResourceManager) HasEnoughResources(Datacenter *mo.Datacenter, Requirements DatacenterResourceRequirements) bool {
	// Returns True if the Datacenter has Enough Resources and meet Customer Requirements

	if _, Error := this.GetComputeResources(Datacenter, Requirements); Error != nil {
		return false
	} else {
		return true
	}
}

func (this *DatacenterResourceManager) GetComputeResources(Datacenter *mo.Datacenter, Requirements DatacenterResourceRequirements) (map[string]object.Reference, error) {
	// Returns Components of the Datacenter, filtered the Requirements
	// Such as Network, Datastore, etc... that will be potentially used to deploy the VM Server

	var ResourceError = errors.New("This Datacenter does not have enough resources to deploy new VM Server with this Configuration, try a bit later.")
	var Resources map[string]object.Reference

	if NetworkResources := NewNetworkResourceManager(this.Client).GetAvailableResources(Datacenter, &Requirements.NetworkResourceRequirements); len(NetworkResources) != 0 {
		Resources["Network"] = NetworkResources[0]
	} else {
		return make(map[string]object.Reference), ResourceError
	}

	if DatastoreResources := NewDatastoreResourceManager(this.Client).GetAvailableResources(Datacenter, Requirements.DatastoreResourceRequirements); len(DatastoreResources) != 0 {
		Resources["Datastore"] = DatastoreResources[0]
	} else {
		return make(map[string]object.Reference), ResourceError
	}

	if StorageResources := NewStorageResourceManager(this.Client).GetAvailableResources(Datacenter, Requirements.StorageResourceRequirements); len(StorageResources) != 0 {
		Resources["Storage"] = StorageResources[0]
	} else {
		return make(map[string]object.Reference), ResourceError
	}

	if ClusterComputeResources := NewClusterComputeResourceManager(this.Client).GetAvailableResources(Datacenter, Requirements.ClusterResourceRequirements); len(ClusterComputeResources) != 0 {
		Resources["ClusterComputeResource"] = ClusterComputeResources[0]
	} else {
		return make(map[string]object.Reference), ResourceError
	}

	if HostSystemResources := NewHostSystemResourceManager(*this.Client).GetAvailableResources(Datacenter, Requirements.HostSystemResourceRequirements); len(HostSystemResources) != 0 {
		Resources["HostSystem"] = HostSystemResources[0]
	} else {
		return make(map[string]object.Reference), ResourceError
	}
	return Resources, nil
}

func (this *DatacenterResourceManager) GetAvailableDatacenters(Requirements DatacenterResourceRequirements) []*object.Datacenter {
	// Method Returns All Available Datacenters, depending on Customer Requirements ...

	// Filtering the Datacenter resources, and making sure, that they are compatible with
	// Customer Requirements

	// NOTE: Every of the Component Upper (Network, Storage, Datastore, etc....)
	// Is being checked within ONLY this Specific Datacenter, that Customer Decided to Pick up
	// If there is not enough resources at this Datacenter, this method would return `False``

	var Datacenters []*object.Datacenter
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*1)
	defer CancelFunc()

	Finder := find.NewFinder(this.Client)
	DatacenterRefs, FindError := Finder.DatacenterList(TimeoutContext, "*")
	Collector := property.DefaultCollector(this.Client)

	if FindError != nil {
		ErrorLogger.Printf("Failed to Find Datacenters, Error: %s", FindError)
		return Datacenters
	}

	for _, Datacenter := range DatacenterRefs {

		var MoDatacenter mo.Datacenter
		CollectError := Collector.RetrieveOne(TimeoutContext, Datacenter.Reference(), []string{"*"}, &MoDatacenter)
		if CollectError != nil {
			DebugLogger.Printf("Failed to Get Mo Interface of the Datacenter, Error: %s", CollectError)
			continue
		}

		if HasEnoughResources := this.HasEnoughResources(&MoDatacenter, Requirements); HasEnoughResources != false {
			Datacenters = append(Datacenters, Datacenter)
		}
	}
	return Datacenters
}

type NetworkResourceManager struct {
	ResourceManagerInterface
	Client vim25.Client
}

func NewNetworkResourceManager(Client *vim25.Client) *NetworkResourceManager {
	return &NetworkResourceManager{
		Client: *Client,
	}
}

func (this *NetworkResourceManager) HasEnoughResources(Network *mo.Network, Requirements *NetworkResourceRequirements) bool {
	// Checks if the Network Has Enough Resources...
	if Network.Summary.GetNetworkSummary().Accessible {
		return true
	} else {
		return false
	} 
}

func (this *NetworkResourceManager) GetAvailableResources(Datacenter *mo.Datacenter, Requirements *NetworkResourceRequirements) []*object.Network {
	// Returns Available Networks, depending on the Resource Requirements...

	var Networks []*object.Network
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*20)
	defer CancelFunc()

	// Checking if Customer Has Specified Any Configuration for the Private Network 
	// If so, instead of picking up existing one from the Public List, we are going to craete 
	// a new one, and make it Isolated from others

	if !reflect.ValueOf(Requirements.Private).IsNil() {
		// Creating New Private Network if Customer Decided to Create One
		Manager := view.NewManager(&this.Client)
		Network, NetworkError := Manager.CreateContainerView(TimeoutContext,
		this.Client.ServiceContent.RootFolder, []string{"Network"}, false) 

		if NetworkError != nil {ErrorLogger.Printf(
		"Failed to Initialize New Network, Error: %s", NetworkError); return Networks}

		Networks = append(Networks, object.NewReference(
		&this.Client, Network.Reference()).(*object.Network))
	}

	// if the Customer has chosen the Public Network, looking for the Public One 
	// that matches specified Requirements 

	for _, Network := range Datacenter.Network {
		var MoNetwork mo.Network
		Collector := property.DefaultCollector(&this.Client)
		CollectError := Collector.RetrieveOne(TimeoutContext, Network.Reference(), []string{"*"}, &MoNetwork)

		if CollectError != nil {
			ErrorLogger.Printf("Failed to Collect Network Resource, Error: %s", CollectError)
			continue
		}
		if HasEnough := this.HasEnoughResources(&MoNetwork, Requirements); HasEnough != false {
			Networks = append(Networks, object.NewReference(&this.Client, Network).(*object.Network))
		}
	}
	return Networks
}

type DatastoreResourceManager struct {
	ResourceManagerInterface
	Client vim25.Client
}

func NewDatastoreResourceManager(Client *vim25.Client) *DatastoreResourceManager {
	return &DatastoreResourceManager{
		Client: *Client,
	}
}

func (this *DatastoreResourceManager) HasEnoughResources(Datastore *mo.Datastore, Requirements DatastoreResourceRequirements) bool {
	// Returns True if the Datastore Has enough Resources as provided in the Requirements

	// Checking If Datastore Has Enough Capacity
	if !(Datastore.Summary.Capacity >= Requirements.Capacity) {
		DebugLogger.Printf("Datastore with Name: `%s` Does not Have Enough Capacity, according to the Resource Requirements", Datastore.Name)
		return false
	}
	// Checking if Datastore Is Accessible
	if !(Datastore.Summary.Accessible) {
		DebugLogger.Printf("Datastore with Name: `%s` is not accessible", Datastore.Name)
		return false
	}
	// Checking if Datastore has enough Free Space, to Run the Customer Application
	if !(Datastore.Summary.FreeSpace >= int64(Requirements.FreeSpace)) {
		DebugLogger.Printf("Datastore with Name: `%s` does not Have Enough Free space, according to Resource Requirements", Datastore.Name)
		return false
	}
	return true
}

func (this *DatastoreResourceManager) GetAvailableResources(Cluster *mo.Datacenter, Requirements DatastoreResourceRequirements) []*object.Datastore {
	// Method Returns List of Available Datastores of the Datacenter, depending on the Requirements
	var Resources []*object.Datastore
	Collector := property.DefaultCollector(&this.Client)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	defer CancelFunc()

	for _, Datastore := range Cluster.Datastore {
		var MoDataStore mo.Datastore
		RetrieveError := Collector.RetrieveOne(TimeoutContext, Datastore, []string{"*"}, MoDataStore)
		if IsEnough := this.HasEnoughResources(&MoDataStore, Requirements); IsEnough == true {
			Resources = append(Resources, object.NewReference(&this.Client, Datastore).(*object.Datastore))
		} else {
			continue
		}

		if RetrieveError != nil {
			DebugLogger.Printf(
				"Failed to Retrieve Datastore, Error: %s", RetrieveError)
		}
	}
	return Resources
}

type StorageResourceManager struct {
	ResourceManagerInterface
	Client vim25.Client
}

func NewStorageResourceManager(Client *vim25.Client) *StorageResourceManager {
	return &StorageResourceManager{
		Client: *Client,
	}
}

func (this *StorageResourceManager) HasEnoughResources(StoragePod *mo.StoragePod, Requirements StorageResourceRequirements) bool {
	// Returns True if the Datastore Has enough Resources as provided in the Requirements

	// Checking if Storage Has Enough Capacity
	if !(StoragePod.Summary.Capacity >= Requirements.Capacity) {
		DebugLogger.Printf("Storage Pod with Name `%s` Does not Have Enough Capacity", StoragePod.Name)
		return false
	}
	// Checking if Storage has enough Free Space
	if !(StoragePod.Summary.FreeSpace >= int64(Requirements.FreeSpace)) {
		DebugLogger.Printf("Storage Pod with Name: `%s` does not Have Enough FreeSpace to Perform an Action", StoragePod.Name)
		return false
	}
	return true
}

func (this *StorageResourceManager) GetAvailableResources(Datacenter *mo.Datacenter, Requirements StorageResourceRequirements) []*object.StoragePod {
	// Method Returns List of Available Datastores of the Datacenter, depending on the Requirements
	var StorageResources []*object.StoragePod
	Collector := property.DefaultCollector(&this.Client)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	defer CancelFunc()

	Finder := find.NewFinder(&this.Client)
	StoragePods, FindError := Finder.DatastoreClusterList(TimeoutContext, "*")

	if FindError != nil {
		return StorageResources
	}
	for _, Storage := range StoragePods {
		var MoStoragePod mo.StoragePod
		RetrieveError := Collector.RetrieveOne(TimeoutContext, Storage.Reference(), []string{"*"}, MoStoragePod)
		if IsEnough := this.HasEnoughResources(&MoStoragePod, Requirements); IsEnough == true {
			StorageResources = append(StorageResources, object.NewReference(&this.Client, Storage.Reference()).(*object.StoragePod))
		} else {
			continue
		}

		if RetrieveError != nil {
			DebugLogger.Printf(
				"Failed to Retrieve Datastore, Error: %s", RetrieveError)
		}
	}
	return StorageResources
}

type ClusterComputeResourceManager struct {
	ResourceManagerInterface
	Client vim25.Client
}

func NewClusterComputeResourceManager(Client *vim25.Client) *ClusterComputeResourceManager {
	return &ClusterComputeResourceManager{
		Client: *Client,
	}
}

func (this *ClusterComputeResourceManager) HasEnoughResources(ClusterComputeResource *mo.ClusterComputeResource, Requirements ClusterComputeResourceRequirements) bool {
	// Returns True if the Datastore Has enough Resources as provided in the Requirements

	// Checking if Total Number Of CPU's is enough, to deploy Customer Application
	if ClusterComputeResource.Summary.GetComputeResourceSummary().TotalCpu < Requirements.CpuNum {
		return false
	}
	// Checking if Total Memory in Megabytes is enough, to deploy customer Application
	if ClusterComputeResource.Summary.GetComputeResourceSummary().TotalMemory*1024 < Requirements.MemoryInMegabytes {
		return false
	}
	return true
}

func (this *ClusterComputeResourceManager) GetAvailableResources(Datacenter *mo.Datacenter, Requirements ClusterComputeResourceRequirements) []*object.ClusterComputeResource {
	// Method Returns List of Available Datastores of the Datacenter, depending on the Requirements

	var ClusterComputeResources []*object.ClusterComputeResource
	Collector := property.DefaultCollector(&this.Client)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	defer CancelFunc()

	Finder := find.NewFinder(&this.Client)
	ClusterComputeResourceList, FindError := Finder.ClusterComputeResourceList(
		TimeoutContext, fmt.Sprintf("%s", Datacenter.Name))

	if FindError != nil {
		ErrorLogger.Printf(
			"Failed to Retrieve Cluster Compute Resources, Error : %s", FindError)
		return ClusterComputeResources
	}

	for _, ClusterComputeResource := range ClusterComputeResourceList {

		var MoClusterComputeResource mo.ClusterComputeResource
		RetrieveError := Collector.RetrieveOne(TimeoutContext, ClusterComputeResource.Reference(), []string{"*"}, MoClusterComputeResource)
		if IsEnough := this.HasEnoughResources(&MoClusterComputeResource, Requirements); IsEnough == true {
			ClusterComputeResources = append(ClusterComputeResources, object.NewReference(&this.Client, ClusterComputeResource.Reference()).(*object.ClusterComputeResource))
		} else {
			continue
		}

		if RetrieveError != nil {
			DebugLogger.Printf(
				"Failed to Retrieve Datastore, Error: %s", RetrieveError)
		}
	}
	return ClusterComputeResources
}

type HostSystemResourceManager struct {
	ResourceManagerInterface
	Client vim25.Client
}

func NewHostSystemResourceManager(Client vim25.Client) *HostSystemResourceManager {
	return &HostSystemResourceManager{
		Client: Client,
	}
}

func (this *HostSystemResourceManager) HasEnoughResources(HostSystem *mo.HostSystem, HostSystemRequirements HostSystemResourceRequirements) bool {
	// Checks if Folder Entity is fullfilling the Requirements
	return true
}
func (this *HostSystemResourceManager) GetAvailableResources(Datacenter *mo.Datacenter, HostSystemRequirements HostSystemResourceRequirements) []*object.HostSystem {
	// Method Returns List of Available Host Systems of the specific Datacenter, depending on the Requirements
	var HostSystemResources []*object.HostSystem
	Collector := property.DefaultCollector(&this.Client)

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	defer CancelFunc()

	Finder := find.NewFinder(&this.Client)
	HostSystems, FindError := Finder.HostSystemList(TimeoutContext, "*")

	if FindError != nil {
		return HostSystemResources
	}

	for _, HostSystem := range HostSystems {
		var MoHostSystem mo.HostSystem
		RetrieveError := Collector.RetrieveOne(TimeoutContext, HostSystem.Reference(), []string{"*"}, MoHostSystem)
		if IsEnough := this.HasEnoughResources(&MoHostSystem, HostSystemRequirements); IsEnough == true {
			HostSystemResources = append(HostSystemResources, object.NewReference(&this.Client, HostSystem.Reference()).(*object.HostSystem))
		} else {
			continue
		}

		if RetrieveError != nil {
			DebugLogger.Printf(
				"Failed to Retrieve Datastore, Error: %s", RetrieveError)
		}
	}
	return HostSystemResources
}
