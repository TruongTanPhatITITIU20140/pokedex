package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Read message from server
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Connection closed by server: %v\n", err)
			break
		}
		fmt.Print(string(buffer[:n]))

		// Read player's input
		if !scanner.Scan() {
			fmt.Println("Failed to read input.")
			break
		}
		input := scanner.Text()

		// Send input to server
		_, err = conn.Write([]byte(input + "\n"))
		if err != nil {
			fmt.Printf("Failed to send command to server: %v\n", err)
			break
		}

		// Handle auto mode input
		if strings.HasPrefix(input, "auto") {
			// Parse duration from input
			args := strings.Split(input, " ")
			if len(args) > 1 {
				duration, err := time.ParseDuration(args[1])
				if err == nil {
					time.Sleep(duration + time.Second)
					continue
				}
			}
		}

		// Wait for a moment before sending the next command
		time.Sleep(time.Millisecond * 100)
	}
}
