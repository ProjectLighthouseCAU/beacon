package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ProjectLighthouseCAU/beacon/auth"
	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/ProjectLighthouseCAU/beacon/handler"
	"github.com/ProjectLighthouseCAU/beacon/network"
	"github.com/ProjectLighthouseCAU/beacon/types"

	"github.com/gorilla/websocket"
)

// Endpoint defines a websocket endpoint
type Endpoint struct { // extends BaseEndpoint implements network.Endpoint (reminder for Java-Dev)
	network.BaseEndpoint // extends
	httpServer           *http.Server
	upgrader             websocket.Upgrader
	connectedClients     map[*types.Client]*websocket.Conn
	clientsLock          sync.Mutex
}

var _ network.Endpoint = (*Endpoint)(nil) // implements

// CreateEndpoint initiates the websocket endpoint (blocking call)
func CreateEndpoint(host string, port int, auth auth.Auth, handler *handler.Handler) *Endpoint {

	defer func() { // recover from any panic during initialization and retry
		if r := recover(); r != nil {
			log.Println("Error while creating websocket endpoint: ", r)
			log.Println("Retrying in 3 seconds...")
			time.Sleep(3 * time.Second)
			CreateEndpoint(host, port, auth, handler)
		}
	}()

	ep := &Endpoint{
		BaseEndpoint: network.BaseEndpoint{
			Type:    network.Websocket,
			Auth:    auth,
			Handler: handler,
		},
		httpServer: &http.Server{Addr: fmt.Sprintf("%s:%d", host, port)},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.WebsocketReadBufferSize,
			WriteBufferSize: config.WebsocketWriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // allow websocket connections from all origin domains
			},
		},
		connectedClients: make(map[*types.Client]*websocket.Conn),
	}
	ep.httpServer.Handler = ep.getWebsocketHandler()
	go func() {
		if err := ep.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Panicf("ListenAndServe returned: %v", err)
		}
	}()

	log.Printf("WebSocket Endpoint created: ws://%s:%d", host, port)

	return ep
}

// Close closes the WebSocket Endpoint
func (ep *Endpoint) Close() {
	log.Println("Closing websocket endpoint")
	ep.clientsLock.Lock()
	defer ep.clientsLock.Unlock()
	for client, conn := range ep.connectedClients {
		log.Println("Disconnecting ", client.Ip())
		// TODO: disconnect client gracefully using close-message
		// with code CloseServiceRestart
		ep.disconnectClient(client, conn, true)
	}
	// delete all connected clients for good measure
	ep.connectedClients = make(map[*types.Client]*websocket.Conn)
	log.Println("All clients disconnected")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := ep.httpServer.Shutdown(ctx)
	if err != nil {
		log.Println("Failed to gracefully shutdown websocket endpoint:", err)
		ep.httpServer.Close()
		log.Println("Forcefully closed websocket endpoint")
	}
	log.Println("Websocket endpoint closed")
}

func (ep *Endpoint) disconnectClient(client *types.Client, conn *websocket.Conn, alreadyLocked bool) {
	// TODO: using a re-entrant lock would be much nicer here
	if !alreadyLocked {
		ep.clientsLock.Lock()
		delete(ep.connectedClients, client)
		ep.clientsLock.Unlock()
	}
	client.Disconnect(ep.Handler.GetDirectory())
	conn.Close()
	log.Println("Client disconnected: ", client.Ip())
}

// The websocket handler upgrades HTTP to WebSocket connections, creates a new Client for that connection
// and then reads and decodes received packets into a Request before forwarding to the handler package.
func (ep *Endpoint) getWebsocketHandler() http.HandlerFunc {
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
		conn.SetReadLimit(int64(config.WebsocketReadLimit)) // set the maximum message size -> closes connection if exceeded

		client := types.NewClient(clientIp, getSendFunc(conn))
		ep.clientsLock.Lock()
		ep.connectedClients[client] = conn
		ep.clientsLock.Unlock()

		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				ep.disconnectClient(client, conn, false)
				return
			}

			if messageType != websocket.BinaryMessage {
				response := types.NewResponse().Reid([]byte{0}).Rnum(http.StatusBadRequest).Warning("Non binary-type message received, use websocket binary-type instead").Build()
				client.Send(response)
				// TODO: send close-message: CloseUnsupportedData
				ep.disconnectClient(client, conn, false)
				return
			}
			request := types.Request{}
			_, err = request.UnmarshalMsg(payload)
			if err != nil {
				response := types.NewResponse().Reid(request.REID).Rnum(http.StatusBadRequest).Warning("Could not deserialize request. Please make sure that you are using the Lighthouse-Protocol correctly").Build()
				client.Send(response)
				// TODO: send close-message: CloseInvalidFramePayloadData
				ep.disconnectClient(client, conn, false)
				return
			}

			// authentication and authorization
			if ok, code := ep.Auth.IsAuthorized(client, &request); !ok {
				response := types.NewResponse().Reid(request.REID).Rnum(code).Build()
				client.Send(response)
				// TODO: decide when to disconnect client connection (without any authentication after timeout?)
				// if code == http.StatusUnauthorized {
				// 	disconnectClient()
				// 	return
				// }
				continue
			}
			ep.Handler.HandleRequest(client, &request)
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
			return err
		}

		lock.Lock()
		err = connection.WriteMessage(websocket.BinaryMessage, data)
		lock.Unlock()
		if err != nil {
			connection.Close()
		}
		return err
	}
}
