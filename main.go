package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp" //gói để điều khiển trình duyệt Chrome không có giao diện (headless)
	"github.com/gocolly/colly" //Gói để lấy data
)

// Chỗ này để định nghĩa struct

type Pokemon struct {
	Name   string   `json:"name"`
	Types  []string `json:"types"`
	Number string   `json:"number"`
	Stats  Stats    `json:"stats"`
	Exp    string   `json:"exp"`
}

type Stats struct {
	HP     int `json:"hp"`
	Attack int `json:"attack"`
	Defense int `json:"defense"`
	Speed  int `json:"speed"`
	SpAtk  int `json:"sp_atk"`
	SpDef  int `json:"sp_def"`
}

func fetchPokemonData(ctx context.Context, i int) (Pokemon, error) {
	var pokemon Pokemon
	var numberStr, hpStr, attackStr, defenseStr, speedStr, spAtkStr, spDefStr string

	err := chromedp.Run(ctx,
		chromedp.Navigate(fmt.Sprintf("https://pokedex.org/#/pokemon/%d", i)),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`document.querySelector(".detail-header .detail-national-id").innerText.replace("#", "")`, &numberStr),
		chromedp.Evaluate(`document.querySelector(".detail-panel-header").innerText`, &pokemon.Name),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.detail-types span.monster-type')).map(elem => elem.innerText.toLowerCase())`, &pokemon.Types),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.detail-stats-row span')).filter(span => span.innerText.includes('HP'))[0].nextElementSibling.innerText`, &hpStr),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.detail-stats-row span')).filter(span => span.innerText.includes('Attack'))[0].nextElementSibling.innerText`, &attackStr),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.detail-stats-row span')).filter(span => span.innerText.includes('Defense'))[0].nextElementSibling.innerText`, &defenseStr),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.detail-stats-row span')).filter(span => span.innerText.includes('Speed'))[0].nextElementSibling.innerText`, &speedStr),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.detail-stats-row span')).filter(span => span.innerText.includes('Sp Atk'))[0].nextElementSibling.innerText`, &spAtkStr),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.detail-stats-row span')).filter(span => span.innerText.includes('Sp Def'))[0].nextElementSibling.innerText`, &spDefStr),
	)

	if err != nil {
		return pokemon, fmt.Errorf("failed to extract data for Number %d: %w", i, err)
	}

	pokemon.Number = strings.TrimSpace(numberStr)
	pokemon.Stats.HP, _ = strconv.Atoi(strings.TrimSpace(hpStr))
	pokemon.Stats.Attack, _ = strconv.Atoi(strings.TrimSpace(attackStr))
	pokemon.Stats.Defense, _ = strconv.Atoi(strings.TrimSpace(defenseStr))
	pokemon.Stats.Speed, _ = strconv.Atoi(strings.TrimSpace(speedStr))
	pokemon.Stats.SpAtk, _ = strconv.Atoi(strings.TrimSpace(spAtkStr))
	pokemon.Stats.SpDef, _ = strconv.Atoi(strings.TrimSpace(spDefStr))

	return pokemon, nil
}

func main() {
	// Tạo context với thời gian timeout dài hơn
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 900*time.Second)
	defer cancel()

	var pokemonList []Pokemon

	for i := 1; i <= 150; i++ {
		var pokemon Pokemon
		var err error

		for retry := 0; retry < 3; retry++ {
			pokemon, err = fetchPokemonData(ctx, i)
			if err == nil {
				break
			}
			log.Printf("Retry %d for Pokemon Number %d: %v", retry+1, i, err)
			time.Sleep(2 * time.Second) // Chờ một chút trước khi thử lại
		}

		if err != nil {
			log.Fatalf("Failed to extract data for Number %d: %v", i, err)
		}

		pokemonList = append(pokemonList, pokemon)
		fmt.Printf("Crawled data for Pokemon Number %d\n", i)
	}

	// Tạo collector mới
	c := colly.NewCollector(
		colly.AllowedDomains("bulbapedia.bulbagarden.net"),
	)

	expMap := make(map[string]string)

	c.OnHTML("table.roundy tbody tr:not(:first-child)", func(e *colly.HTMLElement) {
		Number := strings.Trim(e.ChildText("td:nth-child(1)"), "\n ")
		exp := strings.Trim(e.ChildText("td:nth-child(4)"), "\n ") // Cột exp điều chỉnh đúng
		Number = strings.TrimLeft(Number, "0") // Bỏ số 0

		if Number != "" && exp != "" {
			expMap[Number] = exp
		}
	})

	c.Visit("https://bulbapedia.bulbagarden.net/wiki/List_of_Pok%C3%A9mon_by_effort_value_yield_(Generation_IX)")

	for i := range pokemonList {
		if exp, found := expMap[pokemonList[i].Number]; found {
			pokemonList[i].Exp = exp
		}
	}

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Something went wrong:", err)
	})

	pokemonJSON, err := json.MarshalIndent(pokemonList, "", "    ")
	if err != nil {
		fmt.Println("Error encoding Pokemon data to JSON:", err)
		return
	}

	err = os.WriteFile("./pokebat/pokedex.json", pokemonJSON, 0644)
	err = os.WriteFile("./pokecat/pokedex.json", pokemonJSON, 0644)
	if err != nil {
		fmt.Println("Error writing JSON data to file:", err)
		return
	}
	fmt.Println("Pokemon data saved to pokedex.json")
}
