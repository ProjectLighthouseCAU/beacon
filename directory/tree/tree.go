package tree

import (
	"errors"
	"strings"
	"sync"

	directoryPkg "lighthouse.uni-kiel.de/lighthouse-server/directory"
	"lighthouse.uni-kiel.de/lighthouse-server/resource"
	resourceImpl "lighthouse.uni-kiel.de/lighthouse-server/resource/broker"
)

// ### Directory Type ###

var _ directoryPkg.Directory = (*directory)(nil) // directory type implements Directory interface

type directory struct {
	root tree
	lock sync.RWMutex
}

func NewTree() *directory {
	return &directory{
		root: &node{
			entries: make(map[string]tree),
		},
	}
}

// ### Tree types ###

type tree interface {
	isTree()
}

type node struct {
	entries map[string]tree
}

func (n *node) isTree() {} // node implements tree

type leaf struct {
	resource resource.Resource
}

func (l *leaf) isTree() {} // leaf implements tree

// ### Directory implementation ###

// Traverses the tree and returns a directory node given a path that points to a directory
func (d *directory) getDirectory(path []string, createMissingNodes bool) (*node, error) {
	current := d.root
	for i := 0; i < len(path); i++ {
		switch x := current.(type) {
		case *node:
			res, ok := x.entries[path[i]]
			if !ok {
				if createMissingNodes {
					res = &node{
						entries: make(map[string]tree),
					}
					x.entries[path[i]] = res
				} else {
					return nil, errors.New("directory " + path[i] + " not found in " + strings.Join(path, "/"))
				}
			}
			current = res
		case *leaf:
			return nil, errors.New(path[i] + " in " + strings.Join(path, "/") + " is not a directory")
		default:
			return nil, errors.New("DirectoryTree unknown type: This error should not happen")
		}
	}
	n, ok := current.(*node)
	if !ok {
		return nil, errors.New("")
	}
	return n, nil
}

// CreateResource creates a resource given a path while creating missing directories
func (d *directory) CreateResource(path []string) error {
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
	n.entries[path[len(path)-1]] = &leaf{
		resource: resourceImpl.Create(path),
	}
	return nil
}

// DeleteResource deletes a resource or a directory given a path (it also closes deleted resources)
func (d *directory) DeleteResource(path []string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if len(path) == 0 {
		return errors.New("cannot delete root directory")
	}
	n, err := d.getDirectory(path[0:len(path)-1], false)
	if err != nil {
		return err
	}
	t, ok := n.entries[path[len(path)-1]]
	if !ok {
		return errors.New(path[len(path)-1] + " not found in " + strings.Join(path, "/"))
	}
	// close all contained resources and delete the entry
	forEach(t, func(r resource.Resource) { r.Close() })
	delete(n.entries, path[len(path)-1])
	return nil
}

// GetResource returns a resource from the directory given a path
func (d *directory) GetResource(path []string) (resource.Resource, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if len(path) == 0 {
		return nil, errors.New("root directory is not a resource")
	}
	n, err := d.getDirectory(path[0:len(path)-1], false)
	if err != nil {
		return nil, err
	}
	res, ok := n.entries[path[len(path)-1]]
	if !ok {
		return nil, errors.New(path[len(path)-1] + " not found in " + strings.Join(path, "/"))
	}
	l, ok := res.(*leaf)
	if !ok {
		return nil, errors.New(path[len(path)-1] + " is not a resource")
	}
	return l.resource, nil
}

// String outputs the directory tree in a nice format starting from path (path=[] for full tree)
func (d *directory) String(path []string) (string, error) { // TODO: list tree starting from path
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
func (n *node) string(prefixAtLayer []bool) string {
	res := ""
	lastIdx := len(n.entries) - 1
	idx := 0
	for k, v := range n.entries {
		for i := 0; i < len(prefixAtLayer); i++ {
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
		// res += k + "\n"
		switch x := v.(type) {
		case *leaf:
			res += k + "[r]\n"
			// nothing to do
		case *node:
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

func (d *directory) ForEach(f func(resource.Resource)) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	forEach(d.root, f)
}

func forEach(t tree, f func(resource.Resource)) {
	switch x := t.(type) {
	case *node:
		for _, subt := range x.entries {
			forEach(subt, f)
		}
	case *leaf:
		f(x.resource)
	}
}

func (d *directory) List(path []string) (map[string]interface{}, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	n, err := d.getDirectory(path, false)

	if err != nil {
		return nil, err
	}

	return list(n), nil
}

func list(n *node) map[string]interface{} {

	result := make(map[string]interface{})

	for k, v := range n.entries {
		switch x := v.(type) {
		case *leaf:
			result[k] = make(map[string]interface{}) // empty map to indicate a resource
		case *node:
			result[k] = list(x) // recursive map to indicate a directory
		}
	}
	return result
}
