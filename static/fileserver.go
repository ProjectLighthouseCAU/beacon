package static

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"lighthouse.uni-kiel.de/lighthouse-server/config"
)

var (
	webinterfaceRoute = config.GetString("WEBINTERFACE_ROUTE", "/")
	webinterfacePort  = config.GetInt("WEBINTERFACE_PORT", 3001)
	websocketPort     = config.GetInt("WEBSOCKET_PORT", 3000)
)

func StartFileserver() {
	// serve static testing site (only works with websocket endpoint enabled)
	log.Println("Serving static files: " + "http://localhost:" + strconv.Itoa(webinterfacePort) + webinterfaceRoute)
	mux := http.NewServeMux()
	mux.HandleFunc("/websocket-port", GetWebsocketPort)
	mux.Handle(webinterfaceRoute, http.FileServer(http.Dir("./static")))
	serv := &http.Server{
		Addr:    ":" + fmt.Sprint(webinterfacePort),
		Handler: mux,
	}
	go log.Fatal(serv.ListenAndServe())
}

// GET route for the webinterface to get the websocket port
func GetWebsocketPort(w http.ResponseWriter, r *http.Request) {
	log.Println("Sending websocket port: ", []byte(fmt.Sprint(websocketPort)))
	io.WriteString(w, fmt.Sprint(websocketPort))
}
