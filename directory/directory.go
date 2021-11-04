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
	//TODO: GetTree -> return Tree representation to inspect
	String(path []string) string
}
