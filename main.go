package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
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

type PlayerPokemon struct {
    Pokemon
    Level           int     `json:"level"`
    AccumulatedExp  int     `json:"accumulated_exp"`
    EV              float64 `json:"ev"`
}

type Player struct {
    CapturedPokemons []PlayerPokemon `json:"captured_pokemons"`
}

func loadPokedex(filename string) ([]Pokemon, error) {
	var pokedex []Pokemon
	file, err := os.Open(filename)
	if err != nil {
			return nil, err
	}
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)
	json.Unmarshal(byteValue, &pokedex)
	return pokedex, nil
}

func capturePokemon(pokedex []Pokemon) PlayerPokemon {
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(pokedex))
	selectedPokemon := pokedex[index]
	ev := 0.5 + rand.Float64()*0.5
	playerPokemon := PlayerPokemon{
			Pokemon:        selectedPokemon,
			Level:          1,
			AccumulatedExp: 0,
			EV:             ev,
	}
	return playerPokemon
}

func savePlayerPokemonList(filename string, player *Player) error {
	file, err := json.MarshalIndent(player, "", "    ")
	if err != nil {
			return err
	}
	return ioutil.WriteFile(filename, file, 0644)
}

func loadPlayerPokemonList(filename string) (*Player, error) {
	var player Player
	file, err := os.Open(filename)
	if err != nil {
			if os.IsNotExist(err) {
					return &player, nil
			}
			return nil, err
	}
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)
	json.Unmarshal(byteValue, &player)
	return &player, nil
}

func battlePokemon(player *Player, pokedex []Pokemon) {
	if len(player.CapturedPokemons) < 2 {
			fmt.Println("You need at least 2 Pokemons to battle.")
			return
	}

	// Choose two random Pokemons from the player's list
	rand.Seed(time.Now().UnixNano())
	index1 := rand.Intn(len(player.CapturedPokemons))
	index2 := rand.Intn(len(player.CapturedPokemons))

	for index1 == index2 {
			index2 = rand.Intn(len(player.CapturedPokemons))
	}

	pokemon1 := &player.CapturedPokemons[index1]
	pokemon2 := &player.CapturedPokemons[index2]

	fmt.Printf("%s vs %s\n", pokemon1.Name, pokemon2.Name)

	// Simple battle logic: Random winner
	winner := rand.Intn(2)
	var winnerPokemon, loserPokemon *PlayerPokemon
	if winner == 0 {
			winnerPokemon = pokemon1
			loserPokemon = pokemon2
	} else {
			winnerPokemon = pokemon2
			loserPokemon = pokemon1
	}

	// Winner gains experience
	expGained, _ := strconv.Atoi(loserPokemon.Exp)
	winnerPokemon.AccumulatedExp += expGained
	fmt.Printf("%s wins and gains %d experience points!\n", winnerPokemon.Name, expGained)

	// Check for level up
	expNeeded := 100 * (1 << (winnerPokemon.Level - 1)) // Double the exp each level
	if winnerPokemon.AccumulatedExp >= expNeeded {
			winnerPokemon.Level++
			winnerPokemon.AccumulatedExp -= expNeeded

			// Update stats
			winnerPokemon.Stats.HP = int(float64(winnerPokemon.Stats.HP) * (1 + winnerPokemon.EV))
			winnerPokemon.Stats.Attack = int(float64(winnerPokemon.Stats.Attack) * (1 + winnerPokemon.EV))
			winnerPokemon.Stats.Defense = int(float64(winnerPokemon.Stats.Defense) * (1 + winnerPokemon.EV))
			winnerPokemon.Stats.SpAtk = int(float64(winnerPokemon.Stats.SpAtk) * (1 + winnerPokemon.EV))
			winnerPokemon.Stats.SpDef = int(float64(winnerPokemon.Stats.SpDef) * (1 + winnerPokemon.EV))
			fmt.Printf("%s leveled up to level %d!\n", winnerPokemon.Name, winnerPokemon.Level)
	}
}


func main() {
	pokedex, err := loadPokedex("pokedex.json")
	if err != nil {
			log.Fatalf("Error loading pokedex: %v", err)
	}

	player, err := loadPlayerPokemonList("player_pokemon.json")
	if err != nil {
			log.Fatalf("Error loading player pokemon list: %v", err)
	}

	for {
			fmt.Println("Welcome to PokeCat n PokeBat!")
			fmt.Println("1. Capture a Pokemon")
			fmt.Println("2. Battle Pokemons")
			fmt.Println("3. Save and Exit")
			fmt.Print("Enter your choice: ")
			var choice int
			fmt.Scan(&choice)

			switch choice {
			case 1:
					pokemon := capturePokemon(pokedex)
					player.CapturedPokemons = append(player.CapturedPokemons, pokemon)
					fmt.Printf("You captured a %s!\n", pokemon.Name)
			case 2:
					battlePokemon(player, pokedex)
			case 3:
					if err := savePlayerPokemonList("player_pokemon.json", player); err != nil {
							log.Fatalf("Error saving player pokemon list: %v", err)
					}
					fmt.Println("Game saved. Goodbye!")
					return
			default:
					fmt.Println("Invalid choice. Please try again.")
			}
	}
}
