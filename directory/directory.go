package directory

// Directory defines the directory tree for bookkeeping of the resources.
type Directory[T any] interface {
	// Creates a leaf at a given path and creates the parent directories if they don't exist.
	// Returns an error if the leaf already exists or the path is incorrect.
	CreateLeaf(path []string, value T) error

	// Creates an empty directory at a given path and creates the parent directories if they don't exist.
	// Returns an error if the directory already exists or the path is incorrect.
	CreateDirectory(path []string) error

	// Deletes a leaf or directory at a given path.
	// Returns an error if the leaf or directory does not exist.
	Delete(path []string) error

	// Returns a leaf at a given path.
	// Returns an error if the leaf does not exist
	GetLeaf(path []string) (T, error)

	// Returns the directory structure as a pretty printed string
	String(path []string) (string, error)

	// Executes a function on every leaf in the directory.
	// When the provided function returns false, further execution is stopped.
	// When the provided function returns an error, the error is returned and further execution is also stopped.
	ForEach(path []string, f func(path []string, value T) (bool, error)) error

	// Returns the directories entries as a list
	List(path []string) (map[string]any, error)
	// Returns the directories subtree structure as a nested map
	ListRecursive(path []string) (map[string]any, error)

	// Changes the root directory of this directory to the given directories root
	ChRoot(dir Directory[T]) error
	// Returns this directories root (used within ChRoot)
	GetRoot() map[string]any
}
