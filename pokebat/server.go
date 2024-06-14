package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
)

type Pokemon struct {
	Name   string   `json:"name"`
	Types  []string `json:"types"`
	Number string   `json:"number"`
	Stats  Stats    `json:"stats"`
	Exp    string   `json:"exp"`
}

type Stats struct {
	HP      int `json:"hp"`
	Attack  int `json:"attack"`
	Defense int `json:"defense"`
	Speed   int `json:"speed"`
	SpAtk   int `json:"sp_atk"`
	SpDef   int `json:"sp_def"`
}

type Player struct {
	Name    string
	Pokemons []*Pokemon
	Active  *Pokemon
	Conn    net.Conn
}

var elementalMultipliers = map[string]map[string]float64{
	"fire": {
		"grass": 2.0,
		"water": 0.5,
	},
	"water": {
		"fire": 2.0,
		"grass": 0.5,
	},
	"grass": {
		"water": 2.0,
		"fire": 0.5,
	},
}

var autoBattle = false // Default to manual mode

func main() {
	// Load Pokémon data
	file, err := ioutil.ReadFile("pokedex.json")
	if err != nil {
		log.Fatalf("Failed to load pokedex.json: %v", err)
	}

	var pokemons []Pokemon
	err = json.Unmarshal(file, &pokemons)
	if err != nil {
		log.Fatalf("Failed to parse pokedex.json: %v", err)
	}

	// Start server
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	fmt.Println("Server started. Waiting for players...")

	players := make([]*Player, 0, 2)
	for len(players) < 2 {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		player := &Player{
			Conn: conn,
		}
		players = append(players, player)
		fmt.Printf("Player %d has joined.\n", len(players))
	}

	// Allow players to choose game mode
	for _, player := range players {
		player.Conn.Write([]byte("Choose game mode:\n1. Manual\n2. Automatic\nEnter your choice: "))
		modeChoice := make([]byte, 1024)
		n, err := player.Conn.Read(modeChoice)
		if err != nil {
			log.Printf("Failed to read game mode choice: %v", err)
			continue
		}
		if strings.TrimSpace(string(modeChoice[:n])) == "2" {
			autoBattle = true
			break
		}
	}

	// Assign names and let players choose Pokémons
	for i, player := range players {
		player.Conn.Write([]byte(fmt.Sprintf("Enter your name, Player %d: ", i+1)))
		name := make([]byte, 1024)
		n, err := player.Conn.Read(name)
		if err != nil {
			log.Printf("Failed to read player name: %v", err)
			continue
		}
		player.Name = strings.TrimSpace(string(name[:n]))

		for {
			player.Conn.Write([]byte("Choose 3 Pokémon by entering their numbers (separated by space): "))
			choice := make([]byte, 1024)
			n, err = player.Conn.Read(choice)
			if err != nil {
				log.Printf("Failed to read Pokémon choice: %v", err)
				continue
			}
			choices := strings.Fields(string(choice[:n]))

			if len(choices) != 3 {
				player.Conn.Write([]byte("Invalid Pokémon selection. Please select exactly 3 Pokémon.\n"))
				continue
			}

			player.Pokemons = nil
			for _, choice := range choices {
				found := false
				for _, pokemon := range pokemons {
					if pokemon.Number == choice {
						player.Pokemons = append(player.Pokemons, &pokemon)
						found = true
						break
					}
				}
				if !found {
					player.Conn.Write([]byte(fmt.Sprintf("Pokémon with number %s not found. Please try again.\n", choice)))
					player.Pokemons = nil
					break
				}
			}

			if len(player.Pokemons) == 3 {
				player.Active = player.Pokemons[0]
				break
			}
		}
	}

	// Determine turn order based on Pokémon speed
	var firstPlayer, secondPlayer *Player
	if players[0].Active.Stats.Speed > players[1].Active.Stats.Speed {
		firstPlayer = players[0]
		secondPlayer = players[1]
	} else {
		firstPlayer = players[1]
		secondPlayer = players[0]
	}

	firstPlayer.Conn.Write([]byte(fmt.Sprintf("%s, prepare for battle!\n", firstPlayer.Name)))
	secondPlayer.Conn.Write([]byte(fmt.Sprintf("%s, prepare for battle!\n", secondPlayer.Name)))

	// Main game loop
	for {
		if autoBattle {
			autoBattleTurn(firstPlayer, secondPlayer)
		} else {
			playerTurn(firstPlayer, secondPlayer)
			playerTurn(secondPlayer, firstPlayer)
		}
	}
}

func autoBattleTurn(firstPlayer *Player, secondPlayer *Player) {
	for _, player := range []*Player{firstPlayer, secondPlayer} {
		if player.Active.Stats.HP <= 0 {
			switchPokemon(player)
			continue
		}

		damage := attack(player, secondPlayer, rand.Float64() < 0.5)
		secondPlayer.Active.Stats.HP -= damage
		fmt.Printf("%s dealt %d damage!\n", player.Name, damage)
		player.Conn.Write([]byte(fmt.Sprintf("You dealt %d damage!\n", damage)))
		secondPlayer.Conn.Write([]byte(fmt.Sprintf("You received %d damage!\n", damage)))

		if secondPlayer.Active.Stats.HP <= 0 {
			secondPlayer.Conn.Write([]byte("Your Pokémon fainted!\n"))
			if checkAllPokemonFainted(secondPlayer) {
				player.Conn.Write([]byte("You win!\n"))
				secondPlayer.Conn.Write([]byte("You lose!\n"))
				return
			}
			switchPokemon(secondPlayer)
		}

		// Switch turns
		firstPlayer, secondPlayer = secondPlayer, firstPlayer
		time.Sleep(1 * time.Second) // Add delay to simulate turn
	}
}

func playerTurn(attacker *Player, defender *Player) {
	attacker.Conn.Write([]byte(fmt.Sprintf("Active Pokémon: %v\n", attacker.Active)))
	attacker.Conn.Write([]byte("Choose action:\n1. Attack\n2. Switch Pokémon\nEnter your choice: "))

	choice := make([]byte, 1024)
	n, err := attacker.Conn.Read(choice)
	if err != nil {
		log.Printf("Failed to read player choice: %v", err)
		return
	}

	switch strings.TrimSpace(string(choice[:n])) {
	case "1":
		damage := attack(attacker, defender, rand.Float64() < 0.5)
		defender.Active.Stats.HP -= damage
		attacker.Conn.Write([]byte(fmt.Sprintf("You dealt %d damage!\n", damage)))
		defender.Conn.Write([]byte(fmt.Sprintf("You received %d damage!\n", damage)))

		if defender.Active.Stats.HP <= 0 {
			defender.Conn.Write([]byte("Your Pokémon fainted!\n"))
			if checkAllPokemonFainted(defender) {
				attacker.Conn.Write([]byte("You win!\n"))
				defender.Conn.Write([]byte("You lose!\n"))
				return
			}
			switchPokemon(defender)
		}
	case "2":
		switchPokemon(attacker)
	default:
		attacker.Conn.Write([]byte("Invalid choice. Try again.\n"))
	}
}

func attack(attacker *Player, defender *Player, isSpecial bool) int {
	attackerStats := attacker.Active.Stats
	defenderStats := defender.Active.Stats

	if !isSpecial {
		damage := attackerStats.Attack - defenderStats.Defense
		if damage < 0 {
			damage = 0
		}
		return damage
	} else {
		damage := int(float64(attackerStats.SpAtk)*getElementalMultiplier(attacker.Active, defender.Active) - float64(defenderStats.SpDef))
		if damage < 0 {
			damage = 0
		}
		return damage
	}
}

func switchPokemon(player *Player) {
	player.Conn.Write([]byte("Choose a Pokémon to switch to:\n"))
	validChoices := make(map[int]*Pokemon)
	for i, pokemon := range player.Pokemons {
		if pokemon != player.Active && pokemon.Stats.HP > 0 {
			player.Conn.Write([]byte(fmt.Sprintf("%d. %s\n", i, pokemon.Name)))
			validChoices[i] = pokemon
		}
	}

	if len(validChoices) == 0 {
		player.Conn.Write([]byte("No valid Pokémon to switch to!\n"))
		return
	}

	choice := make([]byte, 1024)
	n, err := player.Conn.Read(choice)
	if err != nil {
		log.Printf("Failed to read Pokémon switch choice: %v", err)
		return
	}

	selectedIndex := -1
	fmt.Sscanf(string(choice[:n]), "%d", &selectedIndex)
	if selectedPokemon, ok := validChoices[selectedIndex]; ok {
		player.Active = selectedPokemon
		player.Conn.Write([]byte(fmt.Sprintf("Switched to %v\n", player.Active)))
	} else {
		player.Conn.Write([]byte("Invalid choice. Try again.\n"))
		switchPokemon(player)
	}
}

func getElementalMultiplier(attacker *Pokemon, defender *Pokemon) float64 {
	maxMultiplier := 1.0
	for _, atkType := range attacker.Types {
		for _, defType := range defender.Types {
			if multiplier, ok := elementalMultipliers[atkType][defType]; ok {
				if multiplier > maxMultiplier {
					maxMultiplier = multiplier
				}
			}
		}
	}
	return maxMultiplier
}

func checkAllPokemonFainted(player *Player) bool {
	for _, pokemon := range player.Pokemons {
		if pokemon.Stats.HP > 0 {
			return false
		}
	}
	return true
}
