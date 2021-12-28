package directory

import (
	"github.com/ProjectLighthouseCAU/beacon/resource"
)

// Directory defines the directory tree for bookkeeping of the resources.
type Directory interface {
	// Creates a resource at a given path and creates the parent directories if they don't exist.
	// Returns an error if the resource already exists or the path is incorrect.
	CreateResource(path []string) error

	// Creates an empty directory at a given path and creates the parent directories if they don't exist.
	// Returns an error if the directory already exists or the path is incorrect.
	CreateDirectory(path []string) error

	// Deletes a resource or directory at a given path.
	// Returns an error if the resource or directory does not exist.
	Delete(path []string) error

	// Returns a resource at a given path.
	// Returns an error if the resource does not exist
	GetResource(path []string) (resource.Resource, error)

	// Returns the directory structure as a pretty printed string
	String(path []string) (string, error)

	// Executes a function on every resource in the directory.
	// When the provided function returns false, further execution is stopped.
	// When the provided function returns an error, the error is returned and further execution is also stopped.
	ForEach(f func(resource.Resource) (bool, error)) error

	// Returns the directory structure as a nested map
	List(path []string) (map[string]interface{}, error)
}
