package snapshot

import (
	"log"
	"os"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
	"github.com/ProjectLighthouseCAU/beacon/resource"
)

type Snapshotter struct {
	directory directory.Directory[resource.Resource]
	stop      chan struct{}
	done      chan struct{}
}

func CreateSnapshotter(d directory.Directory[resource.Resource]) *Snapshotter {
	return &Snapshotter{
		directory: d,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

func (s *Snapshotter) Start() {
	go s.snapshotLoop()
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

// Goroutine that takes a snapshot of the entire directory every snapshotInterval
func (s *Snapshotter) snapshotLoop() {
	var f *os.File
	_, err := os.Stat(config.SnapshotPath)
	if err != nil {
		f, err = os.Create(config.SnapshotPath)
		if err != nil {
			log.Println("[ERROR] could not create snapshot file", err)
			return
		}
	} else {
		f, err = os.OpenFile(config.SnapshotPath, os.O_RDWR, 0644)
		if err != nil {
			log.Println("[ERROR] could not open snapshot file", err)
			return
		}
	}
	defer f.Close()
Loop:
	for {
		select {
		case <-s.stop:
			break Loop
		case <-time.After(config.SnapshotInterval):
			// start := time.Now()
			f.Truncate(0)
			f.Seek(0, 0)
			s.directory.Snapshot([]string{}, f)
			// elapsed := time.Since(start)
			// log.Printf("Created snapshot in %s\n", elapsed)
		}
	}
	f.Truncate(0)
	f.Seek(0, 0)
	s.directory.Snapshot([]string{}, f)
	log.Printf("Created snapshot before shutdown")
	close(s.done)
}
