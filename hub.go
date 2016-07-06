package sseserver

import (
	"fmt"
	"strings"
	"time"

	. "github.com/azer/debug"
)

// A connection hub keeps track of all the active client connections, and
// handles broadcasting messages out to those connections that match the
// appropriate namespace.
type hub struct {
	broadcast   chan SSEMessage      // Inbound messages to propogate out.
	connections map[*connection]bool // Registered connections.
	register    chan *connection     // Register requests from the connections.
	unregister  chan *connection     // Unregister requests from connections.
	sentMsgs    uint64               // Msgs broadcast since startup
	startupTime time.Time            // Time hub was created
}

func newHub() *hub {
	return &hub{
		broadcast:   make(chan SSEMessage),
		connections: make(map[*connection]bool),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		startupTime: time.Now(),
	}
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			Debug("new connection being registered for " + c.namespace)
			h.connections[c] = true
		case c := <-h.unregister:
			Debug("connection told us to unregister for " + c.namespace)
			delete(h.connections, c)
			close(c.send)
		case msg := <-h.broadcast:
			h.sentMsgs++
			formattedMsg := msg.sseFormat()
			for c := range h.connections {
				if strings.HasPrefix(msg.Namespace, c.namespace) {
					select {
					case c.send <- formattedMsg:
					default:
						Debug("cant pass to a connection send chan, buffer is full -- kill it with fire")
						delete(h.connections, c)
						close(c.send)
						c.r.Body.Close()
						c.r.ProtoMajor = 1
						c.r.ProtoMinor = 0
						c.w.Header().Set("Connection", "close")
						c.w.Write([]byte(fmt.Sprintf("data:%s\n\n", "closing connection")))
					}
				}
			}
		}
	}
}
