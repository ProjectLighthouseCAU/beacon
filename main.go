// The main package handles incoming websocket connections and decodes received packets with msgpack.
// The decoded packets are forwarded as server.Request to the server package.
package main

import (
	"bufio"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/auth/hardcoded"
	"github.com/ProjectLighthouseCAU/beacon/auth/legacy"
	"github.com/ProjectLighthouseCAU/beacon/directory/tree"
	"github.com/ProjectLighthouseCAU/beacon/handler"
	"github.com/ProjectLighthouseCAU/beacon/network"
	"github.com/ProjectLighthouseCAU/beacon/network/websocket"
	"github.com/ProjectLighthouseCAU/beacon/resource"
	"github.com/ProjectLighthouseCAU/beacon/resource/broker"
	"github.com/ProjectLighthouseCAU/beacon/resource/brokerless"
	"github.com/ProjectLighthouseCAU/beacon/snapshot"
	"github.com/ProjectLighthouseCAU/beacon/static"

	"github.com/ProjectLighthouseCAU/beacon/config"
)

var (
	websocketHost  = config.GetString("WEBSOCKET_HOST", "127.0.0.1")
	websocketPort  = config.GetInt("WEBSOCKET_PORT", 3000)
	websocketRoute = config.GetString("WEBSOCKET_ROUTE", "/websocket")
	// tcpPort        = config.GetInt("TCP_PORT", 3001)
	snapshotPath     = config.GetString("SNAPSHOT_PATH", "./beacon-snapshot")
	snapshotInterval = config.GetDuration("SNAPSHOT_INTERVAL", 1*time.Second)
)

// The main function sets up the webserver routes for websocket connections
// and for the testing site.
func main() {
	// ### PROFILING ###
	// var f *os.File
	// var e error
	// f, e = os.Create("cpuprofile.pprof")
	// if e != nil {
	// 	log.Fatal(e)
	// }
	// pprof.StartCPUProfile(f)
	// defer pprof.StopCPUProfile()

	// f, e = os.Create("memprofile.pprof")
	// if e != nil {
	// 	log.Fatal(e)
	// }
	// defer pprof.WriteHeapProfile(f)

	// ### STARTUP ###
	log.Println("Starting server...")

	log.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))

	var createResourceFunc func(path []string) resource.Resource
	switch config.GetString("RESOURCE_IMPL", "brokerless") {
	case "brokerless":
		createResourceFunc = brokerless.Create
	case "broker":
		createResourceFunc = broker.Create
	default:
		log.Println("RESOURCE_IMPL environment variable not specified, using \"brokerless\" as default")
		createResourceFunc = brokerless.Create
	}

	directory := tree.NewTree(createResourceFunc)

	f, err := os.Open(snapshotPath)
	if err != nil {
		log.Println("could not create or open snapshot file")
	}
	err = directory.Restore([]string{}, f)
	if err != nil {
		log.Println("could not restore snapshot file:", err)
	} else {
		log.Println("Restored state from snapshot")
	}

	var authImpl auth.Auth
	switch config.GetString("AUTH", "") {
	case "hardcoded":
		authImpl = hardcoded.New()
	case "legacy":
		authImpl = legacy.New(directory)
	case "allow_all":
		authImpl = &auth.AllowAll{}
	default:
		log.Println("AUTH environment variable not specified, denying all access by default!")
		authImpl = &auth.AllowNone{}
	}

	handler := handler.New(directory, authImpl)
	handlers := []network.RequestHandler{handler}

	websocketEndpoint := websocket.CreateEndpoint(websocketHost, websocketPort, websocketRoute, handlers)
	endpoints := []network.Endpoint{websocketEndpoint}

	static.StartFileserver()

	log.Println("Server started")

	// wait for interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM) // SIGINT: Ctrl + C, SIGTERM: used by docker

	// TODO: move cli out of main or into new application (maybe add cmd directory)
	// command line input
	reader := bufio.NewReader(os.Stdin)
	stop := make(chan struct{})
	go func() {
	Loop:
		for {
			fmt.Print("[list <path/to/directory>, stop]> ")
			s, err := reader.ReadString('\n')
			s = strings.TrimSuffix(s, "\n")
			if err != nil {
				fmt.Println(err)
				break
			}
			words := strings.Split(s, " ")
			switch words[0] {
			case "stop":
				close(stop)
				break Loop
			case "list":
				path := []string{}
				if len(words) > 1 {
					path = strings.Split(words[1], "/")
				}
				s, err := directory.String(path)
				if err != nil {
					fmt.Println(err.Error())
				}
				fmt.Print(s)
			case "snapshot":
				path := "./beacon-snapshot"
				f, err := os.Create(path)
				if err != nil {
					fmt.Println(err.Error())
				}
				err = directory.Snapshot([]string{}, f)
				if err != nil {
					fmt.Println(err.Error())
				}
				f.Close()
			case "restore":
				path := "./beacon-snapshot"
				f, err := os.Open(path)
				if err != nil {
					fmt.Println(err.Error())
				}
				err = directory.Restore([]string{}, f)
				if err != nil {
					fmt.Println(err.Error())
				}
				f.Close()
			}
		}
	}()
	snapshotter := snapshot.CreateSnapshotter(directory)
	snapshotter.Start()
	log.Printf("Started automatic snapshotting to %s every %s\n", snapshotPath, snapshotInterval)

	// Wait for either interrupt or stop command
	select {
	case <-interrupt:
	case <-stop:
	}

	log.Println("Stopping server...")

	for _, ep := range endpoints {
		ep.Close()
	}
	for _, h := range handlers {
		h.Close()
	}
	log.Println("Closed all endpoints and handlers")
	snapshotter.StopAndWait()
	log.Println("Server stopped")
}
