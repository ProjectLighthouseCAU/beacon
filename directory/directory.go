package directory

import (
	"lighthouse.uni-kiel.de/lighthouse-server/resource"
)

// Directory defines the directory tree for bookkeeping of the resources.
type Directory interface {
	// Creates a resource at a given path, returns an error if the resource already exists or the path is incorrect
	CreateResource(path []string) error
	// Deletes a resource at a given path, returns an error if the resource does not exist
	DeleteResource(path []string) error
	// Returns a resource at a given path, returns an error if the resource does not exist
	GetResource(path []string) (resource.Resource, error)
	// Returns the directory structure as a pretty printed string
	String(path []string) (string, error)
	// Runs a function on every resource in the directory
	ForEach(f func(resource.Resource))
	// Returns the directory structure as a nested map
	List(path []string) (map[string]interface{}, error)
}
