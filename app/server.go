package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
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

	switch {
	case strings.HasPrefix(incomingMessage.Url, "/echo/"):
		result := strings.Split(incomingMessage.Url, "/echo/")[1]
		s.writeTextResponse(conn, 200, result)
	case strings.HasPrefix(incomingMessage.Url, "/files"):
		fileName := strings.Split(incomingMessage.Url, "/files/")[1]
		s.writeFileResponse(conn, 200, fileName)
	case strings.HasPrefix(incomingMessage.Url, "/user-agent"):
		result := incomingMessage.Headers["User-Agent"]
		s.writeTextResponse(conn, 200, result)
	case incomingMessage.Url == "/":
		s.writeTextResponse(conn, 200, "")
	default:
		s.writeTextResponse(conn, 404, "")
	}
}

func (s *Server) writeFileResponse(conn net.Conn, status int, fileName string) {
	var dirFlag string
	response := fmt.Sprintf("HTTP/1.1 %d \r\n", status)
	response += "Content-Type: application/octet-stream\r\n"

	flag.StringVar(&dirFlag, "directory", "", "")
	flag.Parse()

	//if len(dirFlag) == 0 {
	//	conn.Write([]byte(response))
	//	return
	//}
	//
	fileContent, _ := os.ReadFile("/" + dirFlag + "/" + fileName)

	//if err != nil {
	//	conn.Write([]byte(response))
	//	return
	//}
	//
	response += fmt.Sprintf("Content-Length: %d\r\n\r\n", len(fileContent))
	response += string(fileContent)

	conn.Write([]byte(response))
}

func (s *Server) writeTextResponse(conn net.Conn, status int, body string) {
	response := fmt.Sprintf("HTTP/1.1 %d \r\n", status)
	response += "Content-Type: text/plain\r\n"
	response += fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	response += body
	conn.Write([]byte(response))
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
