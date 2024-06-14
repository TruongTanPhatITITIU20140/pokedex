package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("Failed to connect to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Read welcome message from server
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Failed to read welcome message: %v\n", err)
		return
	}
	fmt.Print(string(buffer[:n]))

	// Main loop to handle player's commands
	for {
		// Read message from server
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Connection closed by server: %v\n", err)
			break
		}
		fmt.Print(string(buffer[:n]))

		// Read player's input
		var input string
		fmt.Scanln(&input)

		// Send input to server
		_, err = conn.Write([]byte(input + "\n"))
		if err != nil {
			fmt.Printf("Failed to send command to server: %v\n", err)
			break
		}

		// Wait for a moment before sending the next command
		time.Sleep(time.Millisecond * 100)
	}
}
