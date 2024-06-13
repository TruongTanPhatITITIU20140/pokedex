package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	stdReader := bufio.NewReader(os.Stdin)

	go readServer(reader)

	for {
		input, _ := stdReader.ReadString('\n')
		fmt.Fprint(writer, input)
		writer.Flush()
	}
}

func readServer(reader *bufio.Reader) {
	for {
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading from server: %v", err)
		}
		fmt.Print(response)
	}
}
