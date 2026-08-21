package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"napredni_algoritmi_projekat/internal/config"
	"napredni_algoritmi_projekat/internal/engine"
)

func main() {
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		fmt.Println("Greska pri ucitavanju konfiguracije:", err)
		return
	}

	kvEngine := engine.NewEngine(cfg.MemtableSize)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("KV Engine")
	fmt.Println("Dostupne komande:")
	fmt.Println("PUT <key> <value>")
	fmt.Println("GET <key>")
	fmt.Println("DELETE <key>")
	fmt.Println("EXIT")

	for {
		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := strings.ToUpper(parts[0])

		switch command {
		case "PUT":
			if len(parts) < 3 {
				fmt.Println("Upotreba: PUT <key> <value>")
				continue
			}

			key := parts[1]
			value := strings.Join(parts[2:], " ")

			if err := kvEngine.Put(key, []byte(value)); err != nil {
				fmt.Println("Greska:", err)
				continue
			}

			fmt.Println("OK")

		case "GET":
			if len(parts) != 2 {
				fmt.Println("Upotreba: GET <key>")
				continue
			}

			value, err := kvEngine.Get(parts[1])
			if err != nil {
				fmt.Println("Greska:", err)
				continue
			}

			fmt.Println(string(value))

		case "DELETE":
			if len(parts) != 2 {
				fmt.Println("Upotreba: DELETE <key>")
				continue
			}

			if err := kvEngine.Delete(parts[1]); err != nil {
				fmt.Println("Greska:", err)
				continue
			}

			fmt.Println("OK")

		case "EXIT":
			fmt.Println("Gasenje programa.")
			return

		default:
			fmt.Println("Nepoznata komanda.")
		}
	}
}
