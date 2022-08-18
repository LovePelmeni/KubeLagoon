package suggestions

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

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

type SuggestionsPacker struct {
	// Packing Suggestions into Map
	SuggestManagerInterface
}

func NewSuggestionsPacker() *SuggestionsPacker {
	return &SuggestionsPacker{}
}

func (this *SuggestionsPacker) GetPackedSuggestions(ArrayOfObjRefs []object.Reference) []ResourceSuggestion {

	var Suggestions []ResourceSuggestion
	switch {
	case len(ArrayOfObjRefs) != 0: // if failed to Parse list of available Datastores
		DebugLogger.Printf("No Resources Available for Datastores has been Found")
		return Suggestions

	case len(ArrayOfObjRefs) == 0: // if
		DebugLogger.Printf("Resources has been Parsed, Picking up the Appropriate Ones..")
		group := sync.WaitGroup{}
		for _, ObjectReference := range ArrayOfObjRefs {
			go func() {
				group.Add(1)
				Metadata := map[string]string{ // Metadata About the Virtual Resource

				}
				ResourceSuggestion := NewResourceSuggestion(ObjectReference, Metadata)
				Suggestions = append(Suggestions, *ResourceSuggestion)
				group.Done()
			}()
			group.Wait()
		}
		return Suggestions

	default:
		return Suggestions
	}
}

type NetworkSuggestManager struct {
	SuggestManagerInterface
}

func NewNetworkSuggestManager() *NetworkSuggestManager {
	return &NetworkSuggestManager{}
}

func (this *NetworkSuggestManager) GetResource(ItemID string) (*object.Reference, error) {

}

func (this *NetworkSuggestManager) GetSuggestions() []ResourceSuggestion {

	NewSuggestPacker := NewSuggestionsPacker()
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	Networks, ParseDatastoreError := finder.NetworkList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		Suggestions = NewSuggestPacker.GetPackedSuggestions(Networks)
		return Suggestions
	}
	return Suggestions
}

type DatastoreSuggestManager struct {
	SuggestManagerInterface
}

func NewDatastoreSuggestManager() *DataCenterSuggestManager {
	return &DataCenterSuggestManager{}
}

func (this *DatastoreSuggestManager) GetResource(ItemID string) (*object.Reference, error)

func (this *DatastoreSuggestManager) GetSuggestions() []ResourceSuggestion {

	NewSuggestPacker := NewSuggestionsPacker()
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	Datastores, ParseDatastoreError := finder.DatastoreList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		Suggestions = NewSuggestPacker.GetPackedSuggestions(Datastores)
		return Suggestions
	}
	return Suggestions
}

type DataCenterSuggestManager struct {
	SuggestManagerInterface
}

func NewDataCenterSuggestManager() *DataCenterSuggestManager {
	return &DataCenterSuggestManager{}
}

func (this *DataCenterSuggestManager) GetResource(ItemID string) (*object.Reference, error)

func (this *DataCenterSuggestManager) GetSuggestions() []ResourceSuggestion {

	NewSuggestPacker := NewSuggestionsPacker()
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	DataCenters, ParseDatastoreError := finder.DatacenterList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		Suggestions = NewSuggestPacker.GetPackedSuggestions(DataCenters)
		return Suggestions
	}
	return Suggestions
}

type ResourceSuggestManager struct {
	SuggestManagerInterface
}

func NewResourceSuggestManager() *ResourceSuggestManager {
	return &ResourceSuggestManager{}
}

func (this *ResourceSuggestManager) GetResource(ItemID string) (*object.Reference, error)

func (this *ResourceSuggestManager) GetSuggestions() []ResourceSuggestion {

	NewSuggestPacker := NewSuggestionsPacker()
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	Resources, ParseDatastoreError := finder.ResourcePoolList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		Suggestions = NewSuggestPacker.GetPackedSuggestions(Resources)
		return Suggestions
	}
	return Suggestions
}

type FolderSuggestManager struct {
	SuggestManagerInterface
}

func NewFolderSuggestManager() *FolderSuggestManager {
	return &FolderSuggestManager{}
}

func (this *FolderSuggestManager) GetResource(ItemID string) (*object.Reference, error)

func (this *FolderSuggestManager) GetSuggestions() []ResourceSuggestion {

	NewSuggestPacker := NewSuggestionsPacker()
	TimeoutContext, CancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer CancelFunc()

	finder := find.NewFinder(&vim25.Client{})
	var Suggestions []ResourceSuggestion
	Folders, ParseDatastoreError := finder.FolderList(TimeoutContext, "*")

	switch {
	case ParseDatastoreError != nil:
		return Suggestions

	case ParseDatastoreError == nil:
		Suggestions = NewSuggestPacker.GetPackedSuggestions(Folders)
		return Suggestions
	}
	return Suggestions
}
