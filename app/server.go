package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	quitch     chan struct{}
	msgch      chan []byte
}

type IncomingMessage struct {
	Method  string
	Url     string
	Headers map[string]string
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan []byte, 10),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln
	s.acceptLoop()

	<-s.quitch
	close(s.msgch)

	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("accept error: ", err)
			continue
		}

		go s.readLoop(conn)
	}
}

func (s *Server) readLoop(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 2048)

	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("read error: ", err)
		return
	}

	msg := buf[:n]
	incomingMessage, err := parseIncomingMessage(msg)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 500\r\n\r\n"))
		return
	}

	if strings.HasPrefix(incomingMessage.Url, "/echo/") {
		result := strings.Split(incomingMessage.Url, "/echo/")[1]
		response := "HTTP/1.1 200 OK\r\n"
		response += "Content-Type: text/plain\r\n"
		response += fmt.Sprintf("Content-Length: %d\r\n\r\n", len(result))
		response += result
		conn.Write([]byte(response))
		return
	}

	if strings.HasPrefix(incomingMessage.Url, "/user-agent") {
		result, ok := incomingMessage.Headers["User-Agent"]
		if !ok {
			conn.Write([]byte("HTTP/1.1 500"))
			return
		}
		response := "HTTP/1.1 200 OK\r\n"
		response += "Content-Type: text/plain\r\n"
		response += fmt.Sprintf("Content-Length: %d\r\n\r\n", len(result))
		response += result
		conn.Write([]byte(response))
		return
	}

	if incomingMessage.Url != "/" {
		conn.Write([]byte("HTTP/1.1 404\r\n\r\n"))
		return
	}

	conn.Write([]byte("HTTP/1.1 200\r\n\r\n"))
}

func parseIncomingMessage(message []byte) (IncomingMessage, error) {
	lines := strings.Split(string(message), "\n")

	if len(lines) < 2 {
		fmt.Println("Invalid message format: fewer than 2 lines")
		return IncomingMessage{}, errors.New("not http request")
	}

	firstLine := strings.Split(strings.TrimSpace(lines[0]), " ")

	if len(firstLine) < 2 {
		fmt.Println("Invalid first line format")
		return IncomingMessage{}, errors.New("not http request")
	}

	incomingMessage := IncomingMessage{
		Method:  firstLine[0],
		Url:     firstLine[1],
		Headers: make(map[string]string),
	}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // Skip empty lines
		}
		headerLine := strings.SplitN(line, ":", 2)
		if len(headerLine) >= 2 {
			key := strings.TrimSpace(headerLine[0])
			value := strings.TrimSpace(headerLine[1])
			incomingMessage.Headers[key] = value
		}
	}
	return incomingMessage, nil
}

func main() {
	server := NewServer(":4221")
	fmt.Println("server startred on port:", server.listenAddr)

	log.Fatal(server.Start())
}
