package suggestions

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/LovePelmeni/Infrastructure/exceptions"
	"github.com/vmware/govmomi/find"
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

type ResourceSuggestion struct {
	// Struct, represents Object Info
	Mutex    sync.Mutex
	Object   object.Reference  `json:"Object"`
	Metadata map[string]string `json:"Metadata"`
}

func NewResourceSuggestion(Obj object.Reference, Info map[string]string) *ResourceSuggestion {
	return &ResourceSuggestion{
		Object:   Obj,
		Metadata: Info,
	}
}

type SuggestManagerInterface interface {
	// Default Interface, represents Class, that returns Suggestions about specific
	// Source
	GetSuggestions() []ResourceSuggestion
	GetResource(ItemID string) (*object.Reference, error) // Method, should return specific Object by the Idenitfier
}

type NetworkSuggestManager struct {
	SuggestManagerInterface
	Client vim25.Client
}

func NewNetworkSuggestManager() *NetworkSuggestManager {
	return &NetworkSuggestManager{}
}

func (this *NetworkSuggestManager) GetResource(ItemPath string) (*object.NetworkReference, error) {
	// Method Receiving Instance of the Resource, that Customer has chosen, during Configuration Setup

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := find.NewFinder(&this.Client)
	Resource, FindError := Finder.NetworkList(TimeoutContext, ItemPath)
	switch {
	case FindError == nil:
		return nil, exceptions.ItemDoesNotExist()
	case FindError != nil:
		return &Resource[0], nil
	default:
		return nil, exceptions.ItemDoesNotExist()
	}
}

func (this *NetworkSuggestManager) GetSuggestions() []ResourceSuggestion {
	// Returns Suggested Resources, of different Types, Zones, Unique Names, etc...

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&this.Client)
	var Suggestions []ResourceSuggestion
	Networks, ParseDatastoreError := finder.NetworkList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		// Extracting Metadata From available resources and putting it into a single `Resource Suggestion` Structure
		group := sync.WaitGroup{}
		for _, Network := range Networks {
			go func() {
				group.Add(1)
				Resource := NewResourceSuggestion(Network,
					map[string]string{
						"UniqueName": Network.Reference().Value,
						"Type":       Network.Reference().Type,
						"Path":       Network.GetInventoryPath(),
					})
				Suggestions = append(Suggestions, *Resource)
				group.Done()
			}()
			group.Wait()
			return Suggestions
		}
	}
	return Suggestions
}

type DatastoreSuggestManager struct {
	SuggestManagerInterface
	Client vim25.Client
}

func NewDatastoreSuggestManager() *DataCenterSuggestManager {
	return &DataCenterSuggestManager{}
}

func (this *DatastoreSuggestManager) GetResource(ItemPath string) (*object.Datastore, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := find.NewFinder(&this.Client)
	Datastore, FindError := Finder.Datastore(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return Datastore, nil

	default:
		return Datastore, nil
	}
}

func (this *DatastoreSuggestManager) GetSuggestions() []ResourceSuggestion {

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	Datastores, ParseDatastoreError := finder.DatastoreList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		for _, Datastore := range Datastores {

		}
		return Suggestions
	}
	return Suggestions
}

type DataCenterSuggestManager struct {
	SuggestManagerInterface
	Client *vim25.Client
}

func NewDataCenterSuggestManager() *DataCenterSuggestManager {
	return &DataCenterSuggestManager{}
}

func (this *DataCenterSuggestManager) GetResource(ItemPath string) (*object.Datacenter, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := find.NewFinder(this.Client)
	Datacenter, FindError := Finder.Datacenter(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return Datacenter, nil

	default:
		return Datacenter, nil
	}
}

func (this *DataCenterSuggestManager) GetSuggestions() []ResourceSuggestion {

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	DataCenters, ParseDatastoreError := finder.DatacenterList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:

		for _, Datacenter := range DataCenters {
			Resource := NewResourceSuggestion(
				Datacenter,
				map[string]string{
					"UniqueName": Datacenter.Reference().Value,
					"Type":       Datacenter.Reference().Type,
					"Name":       Datacenter.Name(),
					"Path":       Datacenter.InventoryPath,
				})
			Suggestions = append(Suggestions, *Resource)
		}
	default:
		return Suggestions
	}
	return Suggestions
}

type ResourceSuggestManager struct {
	SuggestManagerInterface
	Client vim25.Client
}

func NewResourceSuggestManager() *ResourceSuggestManager {
	return &ResourceSuggestManager{}
}

func (this *ResourceSuggestManager) GetResource(ItemPath string) (*object.ResourcePool, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := find.NewFinder(&this.Client)
	ResourcePool, FindError := Finder.ResourcePool(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return ResourcePool, nil

	default:
		return ResourcePool, nil
	}
}

func (this *ResourceSuggestManager) GetSuggestions() []ResourceSuggestion {

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	Resources, ParseDatastoreError := finder.ResourcePoolList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		return Suggestions
	}
	return Suggestions
}

type FolderSuggestManager struct {
	SuggestManagerInterface
	Client vim25.Client
}

func NewFolderSuggestManager() *FolderSuggestManager {
	return &FolderSuggestManager{}
}

func (this *FolderSuggestManager) GetResource(ItemPath string) (*object.Folder, error) {
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	Finder := find.NewFinder(&this.Client)
	Folder, FindError := Finder.Folder(TimeoutContext, ItemPath)
	switch {
	case FindError != nil:
		return nil, exceptions.ItemDoesNotExist()

	case FindError == nil:
		return Folder, nil

	default:
		return Folder, nil
	}
}

func (this *FolderSuggestManager) GetSuggestions() []ResourceSuggestion {

	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	Folders, ParseDatastoreError := finder.FolderList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		return Suggestions
	}
	return Suggestions
}
