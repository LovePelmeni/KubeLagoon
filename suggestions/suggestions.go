package suggestions

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"
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
	// Used to filter appropriate Instances of the Resources, depending on this Struct
}

func NewResourceRequirements() *ResourceRequirements {
	return &ResourceRequirements{}
}

type ResourceSuggestion struct {
	// Struct, represents Object Info
	Mutex    sync.Mutex
	Object   *object.Common    `json:"Object"`
	Metadata map[string]string `json:"Metadata"`
}

func NewResourceSuggestion(Obj *object.Common, Info map[string]string) *ResourceSuggestion {
	return &ResourceSuggestion{
		Object:   Obj,
		Metadata: Info,
	}
}

type SuggestManagerInterface interface {
	// Default Interface, represents Class, that returns Suggestions about specific
	// Source
	GetSuggestions() []ResourceSuggestion
	GetResource(ItemPath string) (object.Reference, error) // Method, should return specific Object by the Idenitfier
}

type NetworkSuggestManager struct {
	SuggestManagerInterface
	Client *vim25.Client
}

func NewNetworkSuggestManager(Client vim25.Client) *NetworkSuggestManager {
	return &NetworkSuggestManager{
		Client: &Client,
	}
}

func (this *NetworkSuggestManager) GetResource(ItemPath string) (object.Reference, error) {
	// Method Receiving Instance of the Resource, that Customer has chosen, during Configuration Setup

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := object.NewSearchIndex(this.Client)
	Resource, FindError := Finder.FindByInventoryPath(TimeoutContext, ItemPath)
	switch {
	case FindError == nil:
		return nil, exceptions.ItemDoesNotExist()
	case FindError != nil:
		return Resource, nil
	default:
		return nil, exceptions.ItemDoesNotExist()
	}
}

func (this *NetworkSuggestManager) GetSuggestions(Requirements ResourceRequirements) []ResourceSuggestion {
	// Returns Suggested Resources, of different Types, Zones, Unique Names, etc...
}

type DatastoreSuggestManager struct {
	SuggestManagerInterface
	Client vim25.Client
}

func NewDatastoreSuggestManager(Client vim25.Client) *DataCenterSuggestManager {
	return &DataCenterSuggestManager{
		Client: &Client,
	}
}

func (this *DatastoreSuggestManager) GetResource(ItemPath string) (object.Reference, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := object.NewSearchIndex(&this.Client)
	Datastore, FindError := Finder.FindByInventoryPath(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return Datastore, nil

	default:
		return Datastore, nil
	}
}

func (this *DatastoreSuggestManager) GetSuggestions(Requirements ResourceRequirements) []ResourceSuggestion {
	// Returns Query of the Networks, depending on the Requirements
}

type DataCenterSuggestManager struct {
	SuggestManagerInterface
	Client *vim25.Client
}

func NewDataCenterSuggestManager(Client vim25.Client) *DataCenterSuggestManager {
	return &DataCenterSuggestManager{
		Client: &Client,
	}
}

func (this *DataCenterSuggestManager) GetResource(ItemPath string) (object.Reference, error) {

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := object.NewSearchIndex(this.Client)
	Datacenter, FindError := Finder.FindByInventoryPath(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return Datacenter, nil

	default:
		return Datacenter, nil
	}
}

func (this *DataCenterSuggestManager) GetSuggestions(Requirements ResourceRequirements) []ResourceSuggestion {
	// Returns Suggested Resource Objects, Depending on the Requirements
}

type ResourceSuggestManager struct {
	SuggestManagerInterface
	Client *vim25.Client
}

func NewResourceSuggestManager(Client vim25.Client) *ResourceSuggestManager {
	return &ResourceSuggestManager{
		Client: &Client,
	}
}

func (this *ResourceSuggestManager) GetResource(ItemPath string) (object.Reference, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := object.NewSearchIndex(this.Client)
	ResourcePool, FindError := Finder.FindByInventoryPath(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return ResourcePool, nil

	default:
		return ResourcePool, nil
	}
}

func (this *ResourceSuggestManager) GetSuggestions(Requirements ResourceRequirements) []ResourceSuggestion {

}

func (this *ResourceSuggestManager) GetSuggestionsBasedOnRequirements() {

}

type FolderSuggestManager struct {
	SuggestManagerInterface
	Client *vim25.Client
}

func NewFolderSuggestManager(Client vim25.Client) *FolderSuggestManager {
	return &FolderSuggestManager{
		Client: &Client,
	}
}

func (this *FolderSuggestManager) GetResource(ItemPath string) (object.Reference, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := object.NewSearchIndex(this.Client)
	Folder, FindError := Finder.FindByInventoryPath(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return Folder, nil

	default:
		return Folder, nil
	}
}

func (this *FolderSuggestManager) GetSuggestions(ResourceRequirements ResourceRequirements) []ResourceSuggestion {
	// Returns Available Folders, depending on the Customer Resource Requirements.
}

type ClusterComputeResourceSuggestManager struct {
	Client vim25.Client
}

func NewClusterComputeResourceSuggestManager() *ClusterComputeResourceSuggestManager {
	return &ClusterComputeResourceSuggestManager{}
}

func (this *ClusterComputeResourceSuggestManager) GetResource(ItemPath string) (object.Reference, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()
	SearchIndex := object.NewSearchIndex(&this.Client)
	AvailableCluster, Error := SearchIndex.FindByInventoryPath(TimeoutContext, ItemPath)
	return AvailableCluster, Error
}

func (this *ClusterComputeResourceSuggestManager) GetSuggestions(ResourceRequirements) ([]ResourceSuggestion, error)
