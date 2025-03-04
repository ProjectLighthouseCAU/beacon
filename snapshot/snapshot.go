package snapshot

import (
	"log"
	"os"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/directory"
)

var (
	snapshotPath     = config.GetString("SNAPSHOT_PATH", "./snapshot.beacon")
	snapshotInterval = config.GetDuration("SNAPSHOT_INTERVAL", 1*time.Second)
)

type Snapshotter struct {
	directory directory.Directory
	stop      chan struct{}
	done      chan struct{}
}

func CreateSnapshotter(d directory.Directory) *Snapshotter {
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
	_, err := os.Stat(snapshotPath)
	if err != nil {
		f, err = os.Create(snapshotPath)
		if err != nil {
			log.Println("[ERROR] could not create snapshot file", err)
			return
		}
	} else {
		f, err = os.OpenFile(snapshotPath, os.O_RDWR, 0644)
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
		case <-time.After(snapshotInterval):
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
