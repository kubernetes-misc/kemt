package web

import (
	"bytes"
	"fmt"
	"github.com/foolin/goview"
	"github.com/gorilla/websocket"
	"github.com/kubernetes-misc/kemt/client"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"path"
	"time"
)

func StartServer(listenAddr string) {

	//render page use `page.html` with '.html' will only file template without master layout.
	http.HandleFunc("/kemt/watch", func(w http.ResponseWriter, r *http.Request) {
		err := goview.Render(w, http.StatusOK, "watch", goview.M{})
		if err != nil {
			fmt.Fprintf(w, "Render page.html error: %v!", err)
		}
	})

	//render page use `page.html` with '.html' will only file template without master layout.
	http.HandleFunc("/kemt/log", func(w http.ResponseWriter, r *http.Request) {
		err := goview.Render(w, http.StatusOK, "log", goview.M{})
		if err != nil {
			fmt.Fprintf(w, "Render page.html error: %v!", err)
		}
	})

	http.HandleFunc("/kemt/api/namespaces", handleAPINamespaces)
	http.HandleFunc("/kemt/api/deployments", handleAPIDeployments)
	http.HandleFunc("/kemt/api/pods", handleAPIPods)

	http.HandleFunc("/kemt/static", func(w http.ResponseWriter, r *http.Request) {
		logrus.Println(r.URL.Path)
		filename := "/build/views/static/" + path.Base(r.URL.Path)
		logrus.Println(filename)
		http.ServeFile(w, r, filename)
	})

	//render index use `index` without `.html` extension, that will render with master layout.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := goview.Render(w, http.StatusOK, "index", goview.M{})
		if err != nil {
			fmt.Fprintf(w, "Render index error: %v!", err)
		}
	})

	//render index use `index` without `.html` extension, that will render with master layout.
	http.HandleFunc("/kemt/status", func(w http.ResponseWriter, r *http.Request) {
		err := goview.Render(w, http.StatusOK, "status", goview.M{})
		if err != nil {
			fmt.Fprintf(w, "Render index error: %v!", err)
		}
	})

	fmt.Println("Listening and serving HTTP on", listenAddr)

	hub := newHub()
	go hub.run()

	//fs := http.FileServer(http.Dir("html"))
	//http.Handle("/", fs)

	http.HandleFunc("/kemt/ws", func(w http.ResponseWriter, r *http.Request) {
		namespace := getGetParam(r, "namespace")
		item := getGetParam(r, "item")

		if item == "" {
			c := client.GetEvents(namespace)
			wsClient := serveWs(hub, w, r, func() {
				//handle clean up
			})
			for item := range c {
				wsClient.send <- []byte(item.ToString())
			}
		} else if item == "k8s-status" {
			namespace := getGetParam(r, "namespace")
			stop := false
			wsClient := serveWs(hub, w, r, func() {
				stop = true
			})

			for {
				result := "graph TB\n\n"
				namespaces := make(map[string]string)
				c := client.GetDeploymentsDetail(namespace)
				i := 0
				for d := range c {
					i++
					if d.Status.Replicas == d.Status.AvailableReplicas {
						line := fmt.Sprintf("up%v(%s %v of %v)\n", i, d.Name, d.Status.AvailableReplicas, d.Status.Replicas)
						namespaces[d.Namespace] = namespaces[d.Namespace] + line
					} else {
						line := fmt.Sprintf("down(%s %v of %v)\n", d.Name, d.Status.AvailableReplicas, d.Status.Replicas)
						line += "style down fill:#fbb,stroke:#f66,stroke-width:2px,color:#000,stroke-dasharray: 5, 5\n"
						namespaces[d.Namespace] = namespaces[d.Namespace] + line
					}

				}
				for ns, rows := range namespaces {
					result += "subgraph ns-" + ns + "\n"
					result += rows
					result += "end\n"
				}
				wsClient.send <- []byte(result)

				time.Sleep(30 * time.Second)
				if stop {
					break
				}
			}

		} else {
			c, err, closer := client.GetLogs(namespace, item)
			if err != nil {
				logrus.Errorln("could not get logs")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
				return
			}
			wsClient := serveWs(hub, w, r, func() {
				if err := closer(); err != nil {
					logrus.Errorln("could not close the Kubernetes client connection")
					logrus.Errorln(err)
					return
				}
			})
			for item := range c {
				wsClient.send <- []byte(item)
			}

		}

	})
	logrus.Infoln("listening on", listenAddr)
	err := http.ListenAndServe(listenAddr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.")
}

type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	cleanup func()
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.cleanup()
		//There might be some  number of items in their
		go func() {
			for range c.send {
			}
		}()
		time.Sleep(1 * time.Second)

		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			//// Add queued chat messages to the current websocket message.
			//n := len(c.send)
			//for i := 0; i < n; i++ {
			//	w.Write(newline)
			//	w.Write(<-c.send)
			//}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request, cleanup func()) *Client {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	c := &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		cleanup: cleanup,
	}
	c.hub.register <- c

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go c.writePump()
	go c.readPump()
	return c
}
