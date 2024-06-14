package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

// Configuration constants
const (
	GridSize           = 2000 // Grid size of the world
	MaxPokemonPerBatch = 50    // Max number of Pokémon generated each time
	PokemonDisappear   = 300   // Time in seconds, after which a Pokémon disappears if not caught (60 seconds = 1 minute)
	MaxPokemonCapacity = 200  // Maximum number of Pokémon a player can hold
)

// Pokemon represents the structure of a Pokémon
type Pokemon struct {
	Name          string    `json:"name"`
	Types         []string  `json:"types"`
	Number        string    `json:"number"`
	Stats         struct {
		HP     int `json:"hp"`
		Attack int `json:"attack"`
		Defense int `json:"defense"`
		Speed  int `json:"speed"`
		SpAtk  int `json:"sp_atk"`
		SpDef  int `json:"sp_def"`
	} `json:"stats"`
	Exp           string    `json:"exp"`
	X             int       // X coordinate on the grid
	Y             int       // Y coordinate on the grid
	SpawnTime     time.Time // Spawn time
	DisappearTime time.Time // Disappear time
}

// Player represents a player in the game
type Player struct {
	Name     string
	X        int // X coordinate on the grid
	Y        int // Y coordinate on the grid
	Pokemons []*Pokemon
	Conn     net.Conn
}

var (
	pokemons        []Pokemon
	mutex           sync.Mutex         // Mutex for safe access to shared data
	playerList      []*Player          // Slice to store connected players
	pokemonMap      map[string]Pokemon // Map to store Pokémon based on their position
	disappearChannel chan Pokemon      // Channel to notify about disappearing Pokémon
)

func main() {
	// Load Pokémon data from pokedex.json file
	err := loadPokemonData("pokedex.json")
	if err != nil {
		log.Fatalf("Failed to load Pokémon data: %v", err)
	}

	// Start the server
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	fmt.Println("Server started. Waiting for players...")

	// Initialize map to store Pokémon based on their position
	pokemonMap = make(map[string]Pokemon)

	// Channel to handle Pokémon spawn and disappear notifications
	pokemonChannel := make(chan Pokemon, MaxPokemonPerBatch)
	disappearChannel = make(chan Pokemon)

	// Start routine to generate Pokémon
	go generatePokemon(pokemonChannel)

	// Start routine to handle Pokémon disappear notifications
	go handleDisappear()

	// Accept incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each player connection in a separate goroutine
		go handlePlayer(conn, pokemonChannel)
	}
}

// loadPokemonData loads Pokémon data from a JSON file
func loadPokemonData(filename string) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load Pokémon data file: %v", err)
	}
	err = json.Unmarshal(file, &pokemons)
	if err != nil {
		return fmt.Errorf("failed to parse Pokémon data: %v", err)
	}
	return nil
}

// generatePokemon generates Pokémon continuously and sends them to a channel
func generatePokemon(pokemonChannel chan<- Pokemon) {
	for {
		mutex.Lock()
		// Generate a new Pokemon
		if len(pokemons) == 0 {
			// Reload pokemons from file if there are no more pokemons left in the slice
			if err := loadPokemonData("pokedex.json"); err != nil {
				log.Printf("Failed to reload Pokémon data: %v", err)
				mutex.Unlock()
				continue
			}
		}

		index := rand.Intn(len(pokemons))
		pokemon := pokemons[index]
		key := fmt.Sprintf("%d,%d", rand.Intn(GridSize), rand.Intn(GridSize))
		pokemon.X, pokemon.Y = parsePosition(key)
		pokemon.SpawnTime = time.Now()
		pokemon.DisappearTime = pokemon.SpawnTime.Add(PokemonDisappear * time.Second)

		// Remove the spawned Pokemon from the list or mark as used
		pokemons = append(pokemons[:index], pokemons[index+1:]...)
		pokemonMap[key] = pokemon
		mutex.Unlock()

		// Send the Pokemon through the channel
		pokemonChannel <- pokemon

		fmt.Printf("A wild Pokémon appeared: %s at (%d, %d)\n", pokemon.Name, pokemon.X, pokemon.Y)

		// Schedule disappearance
		go func(p Pokemon) {
			time.Sleep(PokemonDisappear * time.Second)
			disappearChannel <- p
		}(pokemon)

		// Wait for a random duration between 1 to 3 seconds before spawning the next Pokemon
		waitDuration := time.Duration(rand.Intn(3) + 1)
		time.Sleep(waitDuration * time.Second)
	}
}

// handleDisappear handles disappearance notifications for Pokémon
func handleDisappear() {
	for {
		pokemon := <-disappearChannel
		key := fmt.Sprintf("%d,%d", pokemon.X, pokemon.Y)
		fmt.Printf("Pokémon %s at (%d, %d) disappeared\n", pokemon.Name, pokemon.X, pokemon.Y)
		delete(pokemonMap, key)
	}
}

// handlePlayer handles each player's connection
func handlePlayer(conn net.Conn, pokemonChannel <-chan Pokemon) {
	defer conn.Close()

	player := &Player{
		Conn: conn,
		X:    rand.Intn(GridSize),
		Y:    rand.Intn(GridSize),
	}

	mutex.Lock()
	playerList = append(playerList, player)
	mutex.Unlock()

	fmt.Printf("Player connected at (%d, %d)\n", player.X, player.Y)

	player.Conn.Write([]byte(fmt.Sprintf("You are at position (%d, %d)\n", player.X, player.Y)))

	scanner := bufio.NewScanner(conn)
	for {
		player.Conn.Write([]byte("Choose your step: [u][d][r][l] or 'check' to see your Pokémon\n"))

		if !scanner.Scan() {
			fmt.Printf("Player disconnected\n")
			break
		}
		command := strings.TrimSpace(scanner.Text())

		switch command {
		case "s":
			if player.Y > 0 {
				player.Y--
			}
		case "w":
			if player.Y < GridSize-1 {
				player.Y++
			}
		case "a":
			if player.X > 0 {
				player.X--
			}
		case "d":
			if player.X < GridSize-1 {
				player.X++
			}
		case "check":
			player.Conn.Write([]byte("Your Pokémon:\n"))
			for _, p := range player.Pokemons {
				player.Conn.Write([]byte(fmt.Sprintf("- %s\n", p.Name)))
			}
			player.Conn.Write([]byte("End of Pokémon list\n"))
			continue
		default:
			player.Conn.Write([]byte("Invalid command. Try again.\n"))
			continue
		}

		// Check if there is a Pokémon at the player's new position
		mutex.Lock()
		key := fmt.Sprintf("%d,%d", player.X, player.Y)
		if pokemon, exists := pokemonMap[key]; exists && time.Now().Before(pokemon.DisappearTime) {
			// Player catches the Pokémon
			player.Pokemons = append(player.Pokemons, &pokemon)
			fmt.Printf("Player caught Pokémon: %s\n", pokemon.Name)
			player.Conn.Write([]byte(fmt.Sprintf("You caught Pokémon: %s\n", pokemon.Name)))

			// Remove Pokémon from the map and update disappear channel
			delete(pokemonMap, key)
			mutex.Unlock()

			go func(p Pokemon) {
				time.Sleep(PokemonDisappear * time.Second)
				disappearChannel <- p
			}(pokemon)
		} else {
			mutex.Unlock()
		}
		player.Conn.Write([]byte(fmt.Sprintf("Updated position: (%d, %d)\n", player.X, player.Y)))
	}
}

// parsePosition parses a position key into X and Y coordinates
func parsePosition(key string) (int, int) {
	var x, y int
	fmt.Sscanf(key, "%d,%d", &x, &y)
	return x, y
}
