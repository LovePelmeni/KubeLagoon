package suggestions

import "github.com/vmware/govmomi/object"

// Package reprensents set of classes, that represents available options about where you can
// deploy your application

type SuggestManagerInterface interface {
	// Default Interface, represents Class, that returns Suggestions about specific
	// Source
	GetSuggestions() []*object.Common
}

type NetworkSuggestManager struct {
}

type DatastoreSuggestManager struct {
}

type DataCenterSuggestManager struct {
}

type ResourceSuggestManager struct {
}

type FolderSuggestManager struct {
}
