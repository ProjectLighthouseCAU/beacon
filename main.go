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

	"lighthouse.uni-kiel.de/lighthouse-server/auth/legacy"
	"lighthouse.uni-kiel.de/lighthouse-server/directory/tree"
	"lighthouse.uni-kiel.de/lighthouse-server/handler"
	"lighthouse.uni-kiel.de/lighthouse-server/network"
	"lighthouse.uni-kiel.de/lighthouse-server/network/websocket"
	"lighthouse.uni-kiel.de/lighthouse-server/static"

	"lighthouse.uni-kiel.de/lighthouse-server/config"
)

var (
	websocketPort  = config.GetInt("WEBSOCKET_PORT", 3000)
	websocketRoute = config.GetString("WEBSOCKET_ROUTE", "/websocket")
	// tcpPort        = config.GetInt("TCP_PORT", 3001)
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

	// DEPENDENCY INJECTION:
	// auth := &auth.AllowAll{}
	// auth := &auth.AllowNone{}
	auth := legacy.New()

	directory := tree.NewTree()

	handler := handler.New(directory, auth)
	// loggerHandler := handler.NewLogger()
	handlers := []network.RequestHandler{handler}

	websocketEndpoint := websocket.CreateEndpoint(websocketPort, websocketRoute, handlers)
	// tcpEndpoint := tcp.CreateEndpoint(tcpPort, handlers)
	endpoints := []network.Endpoint{websocketEndpoint}

	static.StartFileserver()

	log.Println("Server started")

	// wait for interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM) // SIGINT: Ctrl + C, SIGTERM: used by docker

	// command line input
	reader := bufio.NewReader(os.Stdin)
	stop := make(chan struct{})
	go func() {
	Loop:
		for {
			s, err := reader.ReadString('\n')
			s = strings.TrimSuffix(s, "\n")
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println(s)
			switch s {
			case "stop":
				close(stop)
				break Loop
			case "list":
				fmt.Println(directory.String([]string{}))
			}
		}
	}()

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

	log.Println("Server stopped")
}
