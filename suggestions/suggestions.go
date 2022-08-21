package suggestions

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/LovePelmeni/Infrastructure/converter"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

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
	LogFile, Error := os.OpenFile("Suggestions.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	DebugLogger = log.New(LogFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(LogFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(LogFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	if Error != nil {
		panic(Error)
	}
}

// Package consists of set of classes, that represents available options about "when/where/how" you can
// deploy your application

type ResourceRequirements struct {
	// Resource Requirements Provided by the Customer
	// This Struct Is Being Matched to the Datacenter's Resource Capabilities, in order to find out
	// that Datacenter has enough resources to meet the Customer Requirements
	Requirements interface{} `json:"Requirements" xml:"Requirements"`
}

func NewResourceRequirements(Requirements interface{}) *ResourceRequirements {
	return &ResourceRequirements{
		Requirements: Requirements,
	}
}

type DatacenterSuggestion struct {
	// Struct, represents Datacenter Info
	DatacenterName string `json:"DatacenterName" xml:"DatacenterName"`
	Datastores     int    `json:"Datastores" xml:"Datastores"`
	Networks       int    `json:"Networks" xml:"Networks"`
}

func NewDatacenterSuggestion(Name string, DatastoresCount int, NetworksCount int) *DatacenterSuggestion {
	return &DatacenterSuggestion{
		DatacenterName: Name,
		Datastores:     DatastoresCount,
		Networks:       NetworksCount,
	}
}

type DatacenterSuggestManagerInterface interface {
	// Default Interface, represents Class, that returns Suggestions about specific
	// Source
	GetSuggestions(Requirements ResourceRequirements) []DatacenterSuggestion
}

type DataCenterSuggestManager struct {
	DatacenterSuggestManagerInterface
	Client *vim25.Client
}

func NewDataCenterSuggestManager(Client vim25.Client) *DataCenterSuggestManager {
	return &DataCenterSuggestManager{
		Client: &Client,
	}
}

func (this *DataCenterSuggestManager) GetDatacenterResources(Datacenter *mo.Datacenter, Requirements ResourceRequirements) map[string]*types.ManagedObjectReference {
	// Returns the Map of the Key (name of the Resource) and Value (Resource Instance), based on the Requirements .
	var Resources map[string]*types.ManagedObjectReference

	ResourceManagers := map[string]resources.ResourceManagerInterface{
		"Storage":                resources.StorageResourceManager,
		"Network":                resources.NetworkResourceManager,
		"Datastore":              resources.DatastoreResourceManager,
		"ClusterComputeResource": resources.ClusterComputeResourceManager,
	}
	WaitGroup := sync.WaitGroup{}

	// Filtering the Datacenter resources, and making sure, that they are compatible with
	// Customer Requirements

	// NOTE: Every of the Component Upper (Network, Storage, Datastore, etc....)
	// Is being checked within ONLY this Specific Datacenter, that Customer Decided to Pick up
	// If there is not enough resources at this Datacenter, this method would return `False`

	for ManagerKey, Manager := range ResourceManagers {
		go func() {
			WaitGroup.Add(1)
			if Resource, Error := Manager.GetResource(Datacenter, Requirements); Resource != nil && Error == nil {
				Resources[ManagerKey] = Resource
			} else {
				Resources[ManagerKey] = nil
			}
			WaitGroup.Done()
		}()
		WaitGroup.Wait()
	}
}

func (this *DataCenterSuggestManager) CheckHasEnoughResources(Datacenter *mo.Datacenter, Requirements ResourceRequirements) bool {
	// Checks, that Datacenter has enough resources, based on requirements

	// Filtering the Datacenter resources, and making sure, that they are compatible with
	// Customer Requirements

	// NOTE: Every of the Component Upper (Network, Storage, Datastore, etc....)
	// Is being checked within ONLY this Specific Datacenter, that Customer Decided to Pick up
	// If there is not enough resources at this Datacenter, this method would return `False`

	Resources := this.GetDatacenterResources(Datacenter, Requirements)
	if slices.Contains(maps.Values(Resources), nil) {
		return false
	} else {
		return true
	}
}

func (this *DataCenterSuggestManager) FindAvailableDatacenters(Requirements ResourceRequirements) []types.ManagedObjectReference {
	// Finds Available Resources, that fullfill the Needs of the Client

	var AvailableDatacenters []types.ManagedObjectReference
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := find.NewFinder(this.Client)
	TypeConverter := converter.NewInstanceConverter()
	AllDatacenters, FindError := Finder.DatacenterList(TimeoutContext, "*")

	// Filtering Datacenters, Checking if they meet the Resource Requirements

	for _, Datacenter := range AllDatacenters {

		DatacenterReference := Datacenter.Reference()
		ConvertedDatacenter := TypeConverter.ToMoEntity(&DatacenterReference) // convertes Datacenter of type `object.Datacenter` to Instace of `mo.Datacenter`

		if HasEnoughResources := this.CheckHasEnoughResources(
			ConvertedDatacenter.(*mo.Datacenter), Requirements); HasEnoughResources == true {
			AvailableDatacenters = append(AvailableDatacenters, Datacenter.Reference())
		} else {
			continue
		}
	}

	switch FindError {
	case nil:
		return AvailableDatacenters
	default:
		ErrorLogger.Printf("Failed to Get Available Datacenters, %s", FindError)
		return []types.ManagedObjectReference{}
	}
}

func (this *DataCenterSuggestManager) GetSuggestions(Requirements ResourceRequirements) []DatacenterSuggestion {
	// Returns Suggested Resource Objects, Depending on the Requirements

	var ResponseDatacentersQuerySet []DatacenterSuggestion
	var AvailableDatacenters []mo.Datacenter
	var DatacenterProps = []string{
		"Name",      // Name of the Datacenter
		"Datastore", // Available Datastores, via this Datacenter
		"Network",   // Available Networks, via this Datacenter
	}

	// Receiving Available Datacenters, based on the Customer Needs
	Datacenters := this.FindAvailableDatacenters(Requirements)
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*30)
	defer CancelFunc()

	PropertyCollector := property.DefaultCollector(this.Client)
	//
	FindError := PropertyCollector.Retrieve(
		TimeoutContext, Datacenters, DatacenterProps, &AvailableDatacenters)
	WaitGroup := sync.WaitGroup{}

	// Making Annotations (Converting Datacenters to `DatacenterSuggetion` classes)
	go func() {
		WaitGroup.Add(1)
		for _, Datacenter := range AvailableDatacenters {
			ResourceSuggestion := NewDatacenterSuggestion(Datacenter.Name,
				len(Datacenter.Datastore), len(Datacenter.Network))
			ResponseDatacentersQuerySet = append(ResponseDatacentersQuerySet, *ResourceSuggestion)
		}
		WaitGroup.Done()
	}()

	WaitGroup.Wait()

	switch FindError {
	case nil:
		return ResponseDatacentersQuerySet
	default:
		// If FindError != nil return empty datacenter's suggestions list
		ErrorLogger.Printf("Failed to get Available Datacenters, Error: %s", FindError)
		return []DatacenterSuggestion{}
	}
}
