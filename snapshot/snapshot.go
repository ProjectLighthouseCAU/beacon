package snapshot

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/directory/tree"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/resource/brokerless"
	"github.com/ProjectLighthouseCAU/beacon/types"
	"github.com/tinylib/msgp/msgp"
)

type Snapshotter struct {
	stop chan struct{}
	done chan struct{}
}

func CreateSnapshotter(d directory.Directory[resource.Resource[resource.Content]]) *Snapshotter {
	return &Snapshotter{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (s *Snapshotter) Start(dir directory.Directory[resource.Resource[resource.Content]]) error {
	file, err := openOrCreateFile(config.SnapshotPath)
	if err != nil {
		return fmt.Errorf("[ERROR Snapshotter.Start] cannot open or create snapshot file, running without snapshotter: %w", err)
	}
	go s.snapshotLoop(file, dir)
	return nil
}

// Goroutine that takes a snapshot of the entire directory every config.SnapshotInterval and saves it as a file named config.SnapshotPath
func (s *Snapshotter) snapshotLoop(writer WriteSeekCloser, dir directory.Directory[resource.Resource[resource.Content]]) {
	defer writer.Close()
	for {
		select {
		case <-s.stop:
			defer close(s.done)
			err := snapshot(writer, dir)
			if err != nil {
				log.Println("[ERROR Snapshotter.snapshotLoop] Cannot create snapshot before shutdown:", err)
			} else {
				log.Println("[Snapshotter.snapshotLoop] Created snapshot before shutdown")
			}
			return
		case <-time.After(config.SnapshotInterval):
			err := snapshot(writer, dir)
			if err != nil {
				log.Println("[ERROR Snapshotter.snapshotLoop] Cannot create snapshot:", err)
			}
		}
	}
}

func (s *Snapshotter) StopAndWait() {
	s.Stop()
	s.Wait()
}

func (s *Snapshotter) Stop() {
	s.stop <- struct{}{} // Signal the snapshotter to stop
}

func (s *Snapshotter) Wait() {
	<-s.done // Wait for the snapshotter to finish the last snapshot
}

func openOrCreateFile(filePath string) (*os.File, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0660) // owner: rw- group: rw- other:---
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Only run this when the automatic snapshotter is not running
func Restore(snapshotFilePath string, dir directory.Directory[resource.Resource[resource.Content]]) error {
	file, err := openOrCreateFile(snapshotFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return restore(file, dir)
}

func restore(reader io.ReadSeeker, dir directory.Directory[resource.Resource[resource.Content]]) error {
	reader.Seek(0, io.SeekStart)
	snapshotMsgpack, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	// return new empty directory if snapshot is empty
	if len(snapshotMsgpack) == 0 {
		return nil
	}
	var snapshot types.Snapshot
	bs, err := snapshot.UnmarshalMsg(snapshotMsgpack)
	if err != nil || len(bs) > 0 {
		return err
	}

	newDir := tree.NewTree[resource.Resource[resource.Content]]()
	for pathStr, value := range snapshot {
		path := strings.Split(pathStr, "/")
		content := (resource.Content)(value)
		// special case: msgpack.Nil is decoded as empty array
		// empty arrays are decoded as [0x90] (msgpack array header with length 0)
		if len(content) == 0 {
			content = resource.Nil
		}
		err := newDir.CreateLeaf(path, brokerless.Create(path, content))
		if err != nil {
			return fmt.Errorf("[ERROR snapshot.restore] cannot restore path: %v with value %v: %w", path, value, err)
		}
	}
	// successfully read snapshot into newDir -> delete dir and load snapshot
	dir.ForEach([]string{}, func(path []string, resource resource.Resource[resource.Content]) (bool, error) {
		resource.Close()
		return true, nil
	})
	return dir.ChRoot(newDir)
}

func Snapshot(snapshotFilePath string, dir directory.Directory[resource.Resource[resource.Content]]) error {
	file, err := openOrCreateFile(snapshotFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return snapshot(file, dir)
}

type truncater interface {
	Truncate(size int64) error
}

func snapshot(writer io.WriteSeeker, dir directory.Directory[resource.Resource[resource.Content]]) error {
	start := time.Now()
	// try truncating if supported (e.g. for os.File)
	truncater, ok := writer.(truncater)
	if ok {
		truncater.Truncate(0)
	}
	writer.Seek(0, io.SeekStart)
	snapshot := types.NewSnapshot()
	if err := dir.ForEach([]string{}, func(path []string, value resource.Resource[resource.Content]) (bool, error) {
		// key (path as string)
		// TODO: using []string does not work properly for unmarshaling, since go does not allow slices as map keys
		pathStr := strings.Join(path, "/") // we ensure in handler.go that paths do not contain "/"
		snapshot[pathStr] = (msgp.Raw)(value.Get())
		return true, nil
	}); err != nil {
		return err
	}

	snapshotMsgpack, err := snapshot.MarshalMsg(nil)
	if err != nil {
		return err
	}

	// TODO: remove debug code for msgpack output
	// _, err = msgp.UnmarshalAsJSON(os.Stdout, snapshotMsgpack)
	// if err != nil {
	// 	err := fmt.Errorf("msgp: produced msgpack cannot be unmarshaled, it must be invalid: %w", err)
	// 	log.Println(err)
	// }

	n, err := writer.Write(snapshotMsgpack)
	if err != nil || n != len(snapshotMsgpack) {
		return err
	}
	elapsed := time.Since(start)
	if config.VerboseLogging {
		_ = elapsed
		// log.Println("Created snapshot in:", elapsed.String())
	}
	return nil
}

type WriteSeekCloser interface {
	io.WriteSeeker
	io.Closer
}
