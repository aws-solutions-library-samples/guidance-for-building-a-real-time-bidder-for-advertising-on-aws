package main

import (
	app "load-generator/code"
	"log"
	"os"
)

func main() {
	if err := app.App(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
