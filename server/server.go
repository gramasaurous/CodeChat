// server.go: source code for the CodeChat server
// Authors: Graham Greving, David Taylor, Jake VanAdrighem
// CMPS 112: Final Project - CodeChat

package main

import (
	"encoding/json"
	"errors"
	"os" // for command line args
	"log"
	"net"
	"sync"
)

// Server datatype representing information for a server
type Server struct {
	clients    map[net.Conn]*Client
	numClients int
	// broadcasting channel
	serverChan chan message
	file 	   string
	write      sync.Mutex
}

// Client datatype used to identify a client
// Includes a reference to the parent server, the network connection
// for the client, the user name, and a channel for communicating
// between the client and the server
type Client struct {
	server     *Server
	conn       net.Conn
	name       string
	clientChan chan string // Might be deprecated
}

// internal message passing struct for IPC
// the message contains both the message to be propagated and the
// server's response to the sender.
type message struct {
	client   Client
	msg      OutgoingMessage
	err      error
	res      ClientResponse
	exitflag bool
}

// ClientResponse response message to a client
type ClientResponse struct {
	Cmd         string `json:"cmd"`
	ReturnCmd string `json:"return-cmd"`
	Status 	  bool `json:"status"`
	Payload	  string `json:"payload"`
}

// OutgoingMessage server message passed to all clients
type OutgoingMessage struct {
	Cmd     string `json:"cmd"`
	From    string `json:"from"`
	Payload string `json:"payload"`
}

// write: writes the message to be sent from server to client
// Marshals the OutgoingMessage into a JSON message and
// sends over the given connection
func (msg OutgoingMessage) write(conn net.Conn) error {
	log.Println("writing a outgoing message to", conn.RemoteAddr().String())
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	log.Println("OutgoingMessage: marshalled into JSON")
	n, err := conn.Write(b)
	if n == 0 {
		err = errors.New("OutgoingMessage.write: no bytes written")
	}
	log.Println("OutgoingMessage: written")
	return err
}

// Same as above for client responses
func (res ClientResponse) write(conn net.Conn) error {
	log.Println("writing a client response to", conn.RemoteAddr().String())
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	n, err := conn.Write(b)
	if n == 0 {
		err = errors.New("ClientResponse.write: no bytes written")
	}
	return err
}

// Broadcast: Runs on a thread, propagate messages to the cliens,
// the response to the sender and the outgoing message to the others
func (serv *Server) broadcast() {
	// loop on incoming messages from the server's chan
	for toBroadcast := range serv.serverChan {
		log.Println("Got a message in broadcast")
		// send message to all clients
		i := 0
		// Have to do connections, not clients. Ask Graham
		for conn := range serv.clients {
			// only write the response to the requesting connection
			if conn == toBroadcast.client.conn {
				toBroadcast.res.write(conn)
			} else {
				// write the message to all other connections
				to := serv.clients[conn].name
				log.Println("broadcasting to ", to)
				toBroadcast.msg.write(conn)
				i++
			}
		}
		log.Println("broadcast to ", i, " clients.")
	}
}

// Passed an error, logs the error and returns true or false
// Should be used on an if statement to ensure proper terserverChanmination
// true  -> error
// false -> no error
func checkErr(e error) bool {
	if e != nil {
		log.Println("checkErr:", e)
		return true
	}
	return false
}

// getClients: gets a list of client names. Useful for future additions
func (serv *Server) getClients(conn net.Conn) (string, error) {
	// Builds an array of names, as well as comma separated string
	// just in case we'll need it later
	names := make([]string, serv.numClients)
	i := 0
	var nameStr string
	for _, value := range serv.clients {
		names[i] = value.name
		nameStr += value.name + ", "
		i++
	}
	return nameStr, nil
}

// parseCommand: Parses a command from the client.
// Decodes JSON messages from the client, performs the specified action,
// and generates an appropriate message and returns it. Runs in the
// handle connection thread for a client, parsing commands as they
// come in
func (client *Client) parseCommand(dec *json.Decoder, serv *Server) (message, error) {
	var m message
	var e error
	var msg string
	var cmd string
	var from string
	errRet := true
	m.client = *client
	var v map[string]interface{}
	err := dec.Decode(&v)
	if checkErr(err) {
		return m, err
	}
	from = client.name
	switch v["cmd"] {
	case "connect":
		if name, ok := v["username"]; ok {
			client.name = name.(string)
			msg = serv.file // give the new client the latest copy of the file
			from = client.name
			cmd = "client-connect"
		} else {
			e = errors.New("doCommands: no username passed to connect")
			cmd = "connect"
			errRet = false
		}
	case "rename":
		if newName, ok := v["newname"]; ok {
			msg = client.name
			client.name = newName.(string)
			cmd = "client-rename"
			msg += "," + client.name
		} else {
			e = errors.New("doCommands: no name(s) passed to rename")
			cmd = "rename"
			errRet = false
		}
	case "exit":
		if reason, ok := v["msg"]; ok {
			msg = reason.(string)
		} else {
			msg = "reason unknown"
		}
		cmd = "client-exit"
		m.exitflag = true
	// lots of "msg" this "msg" that. this is a chat message.
	case "msg":
		log.Println("doCommands: got a mesage")
		if message, ok := v["msg"]; ok {
			msg = message.(string)
			cmd = "message"
		} else {
			e = errors.New("doCommands: no message passed to msg")
			cmd = "message"
			errRet = false
		}
	case "update-file":
		if file, ok := v["msg"]; ok {
			cmd = "update-file"
			msg = file.(string)
			serv.file = file.(string)
		}
	default:
		e = errors.New("bad JSON given in doCommands")
		errRet = false
	}
	m.msg = OutgoingMessage{cmd, from, msg}
	// need to fix this errorString
	// don't send the whole file if not necessary.
	if (cmd != "client-connect") {
		msg = ""
	}
	m.res = ClientResponse{"return-status", cmd, errRet, msg}
	m.err = e
	return m, err
}

// Connection Handling
func handleConnection(conn net.Conn, serv *Server) {
	// ensure that the connection is closed before this routine exits
	defer conn.Close()

	log.Println("new connection from " + conn.RemoteAddr().String())

	// Create the client for this connection
	userChan := make(chan string)
	user := &Client{serv, conn, conn.RemoteAddr().String(), userChan}
	serv.clients[conn] = user
	serv.numClients++

	// Create the JSON decoder
	dec := json.NewDecoder(conn)
	for {
		m, err := user.parseCommand(dec, serv)
		serv.serverChan <- m
		if m.exitflag || err != nil {
			// write back to clients
			delete(serv.clients, conn)
			serv.numClients--
			return
		}
	}
}

func main() {
	var port string
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("Need a port")
	} else {
		port = args[0]
	}

	log.Println("CodeChat Server Starting on port " + port)

	// Initialize the server
	serv := new(Server)
	serv.clients = make(map[net.Conn]*Client)
	serv.serverChan = make(chan message)
	serv.file = ""

	// Start the broadcaster
	go serv.broadcast()

	// Set up networking
	ln, err := net.Listen("tcp", ":"+port)
	if checkErr(err) {
		return
	}
	// Handle all incoming connections
	for {
		conn, err := ln.Accept()
		if checkErr(err) {
			break
		}
		go handleConnection(conn, serv)
	}
}
