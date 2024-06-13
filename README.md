# PokeCat n PokeBat Documentation

### This is our Net Centric Project 
```
Truong Tan Phat _ ITITIU20140
Le Cao Nhat Hoang _ ITITIU20205
```

## PokeCat n PokeBat
PokeCat n PokeBat is a text-based Pokemon game written in Go. It features two main modules: PokeCat for capturing Pokemons and PokeBat for battling Pokemons. The game uses TCP connections to facilitate communication between a client and a server. Players can capture Pokemons, battle with them, and save their progress.

## Project Structure
pokedex
- ├── go.mod
- ├── go.sum
- ├── main.go
- ├── client
    - └── client.go
- └── server
   - └── server.go
   - └── WebClawer.go
   - └──pokedex.json
   - └──player_pokemon.json

## Features
**PokeCat:** Capture Pokemons.
**PokeBat:** Battle with captured Pokemons.
**Persistence:** Save and load the game state using JSON files.

## Requirements
Go programming language (version 1.16+)
github.com/chromedp/chromedp for scraping Pokemon data.
github.com/gocolly/colly for additional data fetching.

## Installation & Run
-Clone the Repository
-Initialize Go Modules
-Generate pokedex.json (go run WebClawer.go)
-Start the Server (go run server.go)
-Run the Client (go run client.go)

