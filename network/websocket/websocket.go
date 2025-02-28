package websocket

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/handler"
	"github.com/ProjectLighthouseCAU/beacon/network"
	"github.com/ProjectLighthouseCAU/beacon/types"

	"github.com/gorilla/websocket"
)

var (
	readBufferSize  = config.GetInt("WEBSOCKET_READ_BUFFER_SIZE", 0)
	writeBufferSize = config.GetInt("WEBSOCKET_WRITE_BUFFER_SIZE", 0)
	readLimit       = config.GetInt("WEBSOCKET_READ_LIMIT", 2048)
	certificatePath = config.GetString("TLS_CERT_PATH", "")
	privateKeyPath  = config.GetString("TLS_PRIVATE_KEY_PATH", "")
)

// Endpoint defines a websocket endpoint
type Endpoint struct { // extends BaseEndpoint implements network.Endpoint (reminder for Java-Dev)
	httpServer           *http.Server
	upgrader             websocket.Upgrader
	network.BaseEndpoint // extends
}

var _ network.Endpoint = (*Endpoint)(nil) // implements

// CreateEndpoint initiates the websocket endpoint (blocking call)
func CreateEndpoint(host string, port int, route string, handler *handler.Handler) *Endpoint {

	defer func() { // recover from any panic during initialization and retry
		if r := recover(); r != nil {
			log.Println("Error while creating websocket endpoint: ", r)
			log.Println("Retrying in 3 seconds...")
			time.Sleep(3 * time.Second)
			CreateEndpoint(host, port, route, handler)
		}
	}()

	var tlsEnabled bool = certificatePath != "" && privateKeyPath != ""

	ep := &Endpoint{
		BaseEndpoint: network.BaseEndpoint{
			Type:    network.Websocket,
			Handler: handler,
		},
		httpServer: &http.Server{Addr: host + ":" + strconv.Itoa(port)},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  readBufferSize,
			WriteBufferSize: writeBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // allow websocket connections from all origin domains
			},
		},
	}
	ep.httpServer.Handler = getWebsocketHandler(ep)
	go func() {
		if tlsEnabled {
			if err := ep.httpServer.ListenAndServeTLS(certificatePath, privateKeyPath); err != http.ErrServerClosed {
				log.Panicf("ListenAndServe returned: %v", err)
			}
		} else {
			if err := ep.httpServer.ListenAndServe(); err != http.ErrServerClosed {
				log.Panicf("ListenAndServe returned: %v", err)
			}
		}
	}()

	url := "ws"
	if tlsEnabled {
		url = url + "s"
	}
	url = url + "://" + host + ":" + strconv.Itoa(port) + route
	log.Printf("WebSocket Endpoint created: " + url)

	return ep
}

// Close closes the WebSocket Endpoint
func (ep *Endpoint) Close() {
	ep.httpServer.Close()
	log.Println("Websocket Endpoint closed")
}

// The websocket handler upgrades HTTP to WebSocket connections, creates a new Client for that connection
// and then reads and decodes received packets into a Request before forwarding to the handler package.
func getWebsocketHandler(ep *Endpoint) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Println("Error while handling websocket connection: ", r)
				log.Println("Closing...")
			}
		}()

		clientIp := request.Header.Get("X-Real-Ip")
		if clientIp == "" {
			clientIp = request.Header.Get("X-Forwarded-For")
		}
		if clientIp == "" {
			clientIp = request.RemoteAddr
		}

		log.Printf("Incoming Connection from: %s\n", clientIp)

		conn, err := ep.upgrader.Upgrade(responseWriter, request, nil)
		if err != nil {
			log.Println(err)
			return
		}
		conn.SetReadLimit(int64(readLimit)) // set the maximum message size -> closes connection if exceeded

		client := types.NewClient(getSendFunc(conn))

		disconnectClient := func() {
			client.Disconnect(ep.Handler.GetDirectory())
			conn.Close()
			log.Println("Client disconnected: ", clientIp)
		}

		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				disconnectClient()
				return
			}

			if messageType != websocket.BinaryMessage {
				response := types.NewResponse().Reid([]byte{0}).Rnum(http.StatusBadRequest).Warning("Non binary-type message received, use websocket binary-type instead").Build()
				client.Send(response)
				disconnectClient()
				return
			}
			request := types.Request{}
			_, err = request.UnmarshalMsg(payload)
			if err != nil {
				log.Println(err)
				response := types.NewResponse().Reid(request.REID).Rnum(http.StatusBadRequest).Warning("Could not deserialize request. Please make sure that you are using the Lighthouse-Protocol correctly").Build()
				client.Send(response)
				disconnectClient()
				return
			}

			requestAuthorized := ep.Handler.HandleRequest(client, &request)

			// TODO: only disconnect on 401 not 403!
			if !requestAuthorized {
				disconnectClient()
				return
			}
		}
	}
}

// This function wraps a reference to the websocket connection
// and a mutex lock for synchronous access to that connection into a closure
// and returns a function that takes a server.Response and writes it thread-safe to the websocket connection.
func getSendFunc(connection *websocket.Conn) func(*types.Response) error {

	var lock = &sync.Mutex{}

	return func(response *types.Response) error {
		data, err := response.MarshalMsg(nil)
		if err != nil {
			log.Println(err)
			return err
		}

		lock.Lock()
		err = connection.WriteMessage(websocket.BinaryMessage, data)
		lock.Unlock()
		if err != nil {
			log.Println(err)
			connection.Close()
		}
		return err
	}
}
