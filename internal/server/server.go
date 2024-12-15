package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

// Server ...
type Server struct {
	host string
	port string
}

// Client ...
type Client struct {
	conn net.Conn
}

// Config ...
type Config struct {
	Host string
	Port string
}

// New ...
func New(config *Config) *Server {
	return &Server{
		host: config.Host,
		port: config.Port,
	}
}

// Run ...
func (server *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.host, server.port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		client := &Client{
			conn: conn,
		}
		go client.handleRequest()
	}
}

func (client *Client) handleRequest() {
	reader := bufio.NewReader(client.conn)
	for {
		message, err := reader.ReadByte()
		if err != nil {
			client.conn.Close()
			return
		}
		if message == 161 {
			fmt.Println("New message")
			//go func() {
			//	time.Sleep(time.Second)
			//	client.conn.Write([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 3, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 40, 0, 80, 16, 0, 80, 10, '\n'})
			//}()
		}
		fmt.Printf("%v ", message)
		//client.conn.Write([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 3, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 40, 0, 80, 16, 0, 80, 10})
	}
}
