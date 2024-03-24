package tree

import (
	"encoding/gob"
	"errors"
	"io"
	"strings"
	"sync"

	directoryPkg "github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/tinylib/msgp/msgp"
)

// ### Directory Type ###

var _ directoryPkg.Directory = (*directory)(nil) // directory type implements Directory interface

type directory struct {
	root               tree
	lock               sync.RWMutex
	createResourceFunc func(path []string) resource.Resource
}

func NewTree(createResourceFunc func(path []string) resource.Resource) *directory {
	if createResourceFunc == nil {
		panic("cannot create directory tree without createResourceFunc (nil)")
	}
	return &directory{
		root: &node{
			entries: make(map[string]tree),
		},
		createResourceFunc: createResourceFunc,
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
		resource: d.createResourceFunc(path),
	}
	return nil
}

// CreateDirectory creates a directory given a path while creating missing directories
func (d *directory) CreateDirectory(path []string) error {
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
func (d *directory) Delete(path []string) error {
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
	forEach(t, func(r resource.Resource) (bool, error) {
		r.Close()
		return true, nil
	})
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
func (d *directory) String(path []string) (string, error) {
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
		switch x := v.(type) {
		case *leaf:
			res += k + "[r]\n"
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

// ForEach executes a function on every resource in the directory.
// When the provided function returns false, further execution is stopped.
// When the provided function returns an error, the error is returned and further execution is also stopped.
func (d *directory) ForEach(f func(resource.Resource) (bool, error)) error {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return forEach(d.root, f)
}

func forEach(t tree, f func(resource.Resource) (bool, error)) (err error) {
	switch x := t.(type) {
	case *node:
		for _, subt := range x.entries {
			err = forEach(subt, f)
			if err != nil {
				return
			}
		}
	case *leaf:
		var cont bool
		cont, err = f(x.resource)
		if err != nil || !cont {
			return
		}
	}
	return
}

// List lists the contents of a directory by returning a recursively nested map of subdirectories.
// A resource is indicated by a nil value.
func (d *directory) List(path []string) (map[string]interface{}, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	n, err := d.getDirectory(path, false)
	if err != nil {
		return nil, err
	}
	m, err := list(n, false)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func list(n *node, includeContent bool) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range n.entries {
		switch x := v.(type) {
		case *leaf:
			if includeContent {
				content, _ := x.resource.Get()
				result[k] = ([]byte)(content.(msgp.Raw))
			} else {
				result[k] = nil // nil to indicate a resource (empty map is not distinguishable from empty directory)
			}
		case *node:
			var err error
			result[k], err = list(x, includeContent) // recursive map to indicate a directory
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// Snapshotting might look unnecessarily complicated but there is a good reason for the decisions:
// Marshaling the complete directory and resource contents with MsgPack makes it impossible to disinguish
// between a directory and a map stored inside of a resource.
// We therefore use MsgPack to marshal the resource contents (in order to keep full MsgPack compatibility)
// and then gob (Go's binary encoding) to marshal the directory tree.
// Restoring from a snapshot does the same thing in reverse: First decode with gob and then decode the resources with msgpack.
// Note: shamaton/msgpack library decodes map[string]interface{} as map[interface{}]interface{} -> switched to vmihailenco/msgpack

// Snapshot takes a snapshot of a directory (including the resource contents), serializes and writes it to the io.Writer.
func (d *directory) Snapshot(path []string, writer io.Writer) error {
	d.lock.RLock()
	defer d.lock.RUnlock()
	n, err := d.getDirectory(path, false)
	if err != nil {
		return err
	}
	gob.Register(map[string]interface{}{})
	enc := gob.NewEncoder(writer)
	m, err := list(n, true)
	if err != nil {
		return err
	}
	return enc.Encode(m)
}

// Restore restores a directory from a snapshot provided by the io.Reader.
// It deserializes the snapshot and recreates all directories and resources
// and fills the resources with their values from the snapshot.
func (d *directory) Restore(path []string, reader io.Reader) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	var m map[string]interface{}
	gob.Register(map[string]interface{}{})
	dec := gob.NewDecoder(reader)
	err := dec.Decode(&m)
	if err != nil {
		return err
	}
	return restore(d, path, m)
}

func restore(d *directory, path []string, m map[string]interface{}) error {
	for k, v := range m {
		switch x := v.(type) {
		case map[string]interface{}:
			_, err := d.getDirectory(append(path, k), true)
			if err != nil {
				return err
			}
			err = restore(d, append(path, k), x)
			if err != nil {
				return err
			}
		default:
			n, err := d.getDirectory(path, true)
			if err != nil {
				return err
			}
			_, ok := n.entries[k]
			if ok {
				return errors.New(k + " in " + strings.Join(path, "/") + " already exists")
			}
			r := d.createResourceFunc(append(path, k))
			content := (msgp.Raw)(v.([]byte))
			resp := r.Put(content)
			if resp.Err != nil {
				return err
			}
			n.entries[k] = &leaf{
				resource: r,
			}
		}
	}
	return nil
}
