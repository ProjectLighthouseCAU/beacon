package tree

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	directoryPkg "github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/util"
)

// ### Directory Type ###

var _ directoryPkg.Directory[any] = (*directory[any])(nil) // directory type implements Directory interface

type directory[T any] struct {
	root tree
	lock sync.RWMutex
}

func NewTree[T any]() directoryPkg.Directory[T] {
	return &directory[T]{
		root: &node[T]{
			entries: make(map[string]tree),
		},
	}
}

// ### Tree types ###

type tree interface {
	isTree()
}

type node[T any] struct {
	entries map[string]tree
}

func (n *node[T]) isTree() {} // node implements tree

type leaf[T any] struct {
	// path  []string
	value T
}

func (l *leaf[T]) isTree() {} // leaf implements tree

// ### Directory implementation ###

// TODO: refactor and simplify implementation

// Traverses the tree and returns a directory node given a path that points to a directory
func (d *directory[T]) getDirectory(path []string, createMissingNodes bool) (*node[T], error) {
	current := d.root
	for i := range path {
		switch x := current.(type) {
		case *node[T]:
			res, ok := x.entries[path[i]]
			if !ok {
				if createMissingNodes {
					res = &node[T]{
						entries: make(map[string]tree),
					}
					x.entries[path[i]] = res
				} else {
					return nil, errors.New("directory " + path[i] + " not found in " + strings.Join(path, "/"))
				}
			}
			current = res
		case *leaf[T]:
			return nil, errors.New(path[i] + " in " + strings.Join(path, "/") + " is not a directory")
		default:
			return nil, errors.New("DirectoryTree unknown type: This error should not happen")
		}
	}
	n, ok := current.(*node[T])
	if !ok {
		return nil, errors.New(strings.Join(path, "/") + " is not a directory")
	}
	return n, nil
}

// CreateResource creates a resource given a path while creating missing directories
func (d *directory[T]) CreateLeaf(path []string, value T) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if len(path) == 0 {
		return errors.New("cannot create root directory")
	}
	n, err := d.getDirectory(path[0:len(path)-1], true) // create missing directories in path
	if err != nil {
		return err
	}
	_, ok := n.entries[path[len(path)-1]]
	if ok {
		return errors.New(path[len(path)-1] + " in " + strings.Join(path, "/") + " already exists")
	}
	n.entries[path[len(path)-1]] = &leaf[T]{
		value,
	}
	return nil
}

// CreateDirectory creates a directory given a path while creating missing directories
func (d *directory[T]) CreateDirectory(path []string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if len(path) == 0 {
		return errors.New("cannot create root directory")
	}
	n, _ := d.getDirectory(path, false)
	if n != nil {
		return errors.New("directory " + strings.Join(path, "/") + " already exists")
	}
	_, err := d.getDirectory(path, true) // create missing directories in path
	if err != nil {
		return err
	}
	return nil
}

// DeleteResource deletes a resource or a directory given a path (it also closes deleted resources)
func (d *directory[T]) Delete(path []string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if len(path) == 0 {
		return errors.New("cannot delete root directory")
	}
	n, err := d.getDirectory(path[0:len(path)-1], false)
	if err != nil {
		return err
	}
	_, ok := n.entries[path[len(path)-1]]
	if !ok {
		return errors.New(path[len(path)-1] + " not found in " + strings.Join(path, "/"))
	}
	delete(n.entries, path[len(path)-1])
	return nil
}

// GetResource returns a resource from the directory given a path
func (d *directory[T]) GetLeaf(path []string) (T, error) {
	var emptyValue T
	d.lock.RLock()
	defer d.lock.RUnlock()
	if len(path) == 0 {
		return emptyValue, errors.New("root directory is not a resource")
	}
	n, err := d.getDirectory(path[0:len(path)-1], false)
	if err != nil {
		return emptyValue, err
	}
	res, ok := n.entries[path[len(path)-1]]
	if !ok {
		return emptyValue, errors.New(path[len(path)-1] + " not found in " + strings.Join(path, "/"))
	}
	l, ok := res.(*leaf[T])
	if !ok {
		return emptyValue, errors.New(path[len(path)-1] + " is not a resource")
	}
	return l.value, nil
}

// String outputs the directory tree in a nice format starting from path (path=[] for full tree)
func (d *directory[T]) String(path []string) (string, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	// result := "root\n"
	n, err := d.getDirectory(path, false)
	if err != nil {
		return "", err
	}
	result := ""
	if len(path) == 0 {
		result += "root\n"
	} else {
		result += path[len(path)-1] + "\n"
	}
	result += n.string([]bool{})
	return result, nil
}

// Recursively prints the directory tree
func (n *node[T]) string(prefixAtLayer []bool) string {
	res := ""
	lastIdx := len(n.entries) - 1
	idx := 0
	for k, v := range n.entries {
		for i := range prefixAtLayer {
			if prefixAtLayer[i] {
				res += "│    "
			} else {
				res += "     "
			}
		}
		if idx == lastIdx {
			res += "└── "
		} else {
			res += "├── "
		}
		switch x := v.(type) {
		case *leaf[T]:
			res += k + "[r]\n"
		case *node[T]:
			res += k + "[d]\n"
			if idx == lastIdx {
				res += x.string(append(prefixAtLayer, false))
			} else {
				res += x.string(append(prefixAtLayer, true))
			}
		}
		idx++
	}
	return res
}

// ForEach executes a function on every resource in the directory.
// When the provided function returns false, further execution is stopped.
// When the provided function returns an error, the error is returned and further execution is also stopped.
func (d *directory[T]) ForEach(path []string, f func(path []string, value T) (bool, error)) error {
	l, err := d.GetLeaf(path)
	if err == nil {
		f(path, l)
		return nil
	}
	d.lock.RLock()
	defer d.lock.RUnlock()
	n, err := d.getDirectory(path, false)
	if err != nil {
		return err
	}
	return forEach(n, path, f)
}

func forEach[T any](t tree, path []string, f func(path []string, value T) (bool, error)) (err error) {
	switch x := t.(type) {
	case *node[T]:
		for entryName, subt := range x.entries {
			err = forEach(subt, util.ImmutableAppend(path, entryName), f)
			if err != nil {
				return
			}
		}
	case *leaf[T]:
		var cont bool
		cont, err = f(path, x.value)
		if err != nil || !cont {
			return
		}
	}
	return
}

func (d *directory[T]) List(path []string) ([]string, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	n, err := d.getDirectory(path, false)
	if err != nil {
		return nil, err
	}
	return slices.AppendSeq(make([]string, 0, len(n.entries)), maps.Keys(n.entries)), nil
}

// List lists the contents of a directory by returning a recursively nested map of subdirectories.
// A resource is indicated by a nil value.
func (d *directory[T]) ListRecursive(path []string) (map[string]any, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	n, err := d.getDirectory(path, false)
	if err != nil {
		return nil, err
	}
	m, err := list(n)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func list[T any](n *node[T]) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range n.entries {
		switch x := v.(type) {
		case *leaf[T]:
			result[k] = nil // nil to indicate a resource (empty map is not distinguishable from empty directory)
		case *node[T]:
			var err error
			result[k], err = list(x) // recursive map to indicate a directory
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// Changes the root directory of this directory to the one of the given directory.
// Given directory must not be the same as this directory.
// Given directory must be of same implementation type as this directory.
func (d *directory[T]) ChRoot(dir directoryPkg.Directory[T]) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	newRoot := make(map[string]tree)
	for key, valueIntf := range dir.GetRoot() {
		value, ok := valueIntf.(tree)
		if !ok {
			return fmt.Errorf("[ChRoot] Root directory entry has wrong type, cannot convert from %T to tree", valueIntf)
		}
		newRoot[key] = value
	}
	d.root.(*node[T]).entries = newRoot
	return nil
}

// Returns the root directory of this directory
func (d *directory[T]) GetRoot() map[string]any {
	d.lock.Lock()
	defer d.lock.Unlock()
	root := make(map[string]any)
	for k, v := range d.root.(*node[T]).entries {
		root[k] = v
	}
	return root
}
