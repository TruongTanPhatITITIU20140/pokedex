package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
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
	Mutex            sync.Mutex      `json:"-"`
}

var pokedex []Pokemon

func main() {
	var err error
	pokedex, err = loadPokedex("pokedex.json")
	if err != nil {
		log.Fatalf("Error loading pokedex: %v", err)
	}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error starting TCP server: %v", err)
	}
	defer ln.Close()

	log.Println("Server started on :8080")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection: ", err)
			continue
		}

		player := &Player{}
		go handleConnection(conn, player)
	}
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

func handleConnection(conn net.Conn, player *Player) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	fmt.Fprintln(writer, "Welcome to PokeCat n PokeBat!")
	writer.Flush()

	for {
		fmt.Fprintln(writer, "1. Capture a Pokemon")
		fmt.Fprintln(writer, "2. Battle Pokemons")
		fmt.Fprintln(writer, "3. Save and Exit")
		fmt.Fprint(writer, "Enter your choice: ")
		writer.Flush()

		choiceStr, _ := reader.ReadString('\n')
		choice, err := strconv.Atoi(strings.TrimSpace(choiceStr))
		if err != nil {
			fmt.Fprintln(writer, "Invalid choice. Please try again.")
			writer.Flush()
			continue
		}

		switch choice {
		case 1:
			capturePokemon(player, writer)
		case 2:
			battlePokemon(player, writer)
		case 3:
			savePlayerPokemonList(fmt.Sprintf("player_%s.json", conn.RemoteAddr().String()), player)
			fmt.Fprintln(writer, "Game saved. Goodbye!")
			writer.Flush()
			return
		default:
			fmt.Fprintln(writer, "Invalid choice. Please try again.")
			writer.Flush()
		}
	}
}

func capturePokemon(player *Player, writer *bufio.Writer) {
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

	player.Mutex.Lock()
	player.CapturedPokemons = append(player.CapturedPokemons, playerPokemon)
	player.Mutex.Unlock()

	fmt.Fprintf(writer, "You captured a %s!\n", playerPokemon.Name)
	writer.Flush()
}

func battlePokemon(player *Player, writer *bufio.Writer) {
	player.Mutex.Lock()
	defer player.Mutex.Unlock()

	if len(player.CapturedPokemons) < 2 {
		fmt.Fprintln(writer, "You need at least 2 Pokemons to battle.")
		writer.Flush()
		return
	}

	rand.Seed(time.Now().UnixNano())
	index1 := rand.Intn(len(player.CapturedPokemons))
	index2 := rand.Intn(len(player.CapturedPokemons))

	for index1 == index2 {
		index2 = rand.Intn(len(player.CapturedPokemons))
	}

	pokemon1 := &player.CapturedPokemons[index1]
	pokemon2 := &player.CapturedPokemons[index2]

	fmt.Fprintf(writer, "%s vs %s\n", pokemon1.Name, pokemon2.Name)
	writer.Flush()

	winner := rand.Intn(2)
	var winnerPokemon, loserPokemon *PlayerPokemon
	if winner == 0 {
		winnerPokemon = pokemon1
		loserPokemon = pokemon2
	} else {
		winnerPokemon = pokemon2
		loserPokemon = pokemon1
	}

	expGained, _ := strconv.Atoi(loserPokemon.Exp)
	winnerPokemon.AccumulatedExp += expGained
	fmt.Fprintf(writer, "%s wins and gains %d experience points!\n", winnerPokemon.Name, expGained)
	writer.Flush()

	expNeeded := 100 * (1 << (winnerPokemon.Level - 1))
	if winnerPokemon.AccumulatedExp >= expNeeded {
		winnerPokemon.Level++
		winnerPokemon.AccumulatedExp -= expNeeded

		winnerPokemon.Stats.HP = int(float64(winnerPokemon.Stats.HP) * (1 + winnerPokemon.EV))
		winnerPokemon.Stats.Attack = int(float64(winnerPokemon.Stats.Attack) * (1 + winnerPokemon.EV))
		winnerPokemon.Stats.Defense = int(float64(winnerPokemon.Stats.Defense) * (1 + winnerPokemon.EV))
		winnerPokemon.Stats.SpAtk = int(float64(winnerPokemon.Stats.SpAtk) * (1 + winnerPokemon.EV))
		winnerPokemon.Stats.SpDef = int(float64(winnerPokemon.Stats.SpDef) * (1 + winnerPokemon.EV))
		fmt.Fprintf(writer, "%s leveled up to level %d!\n", winnerPokemon.Name, winnerPokemon.Level)
		writer.Flush()
	}
}

func savePlayerPokemonList(filename string, player *Player) error {
	player.Mutex.Lock()
	defer player.Mutex.Unlock()

	file, err := json.MarshalIndent(player, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, file, 0644)
}
