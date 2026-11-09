package main

import (
	"log"
	"os"

	"atonixcorp/cloud/devtools/portal/backend"
)

func main() {
	addr := os.Getenv("DEVTOOLS_BACKEND_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	if err := backend.Start(addr); err != nil {
		log.Fatal(err)
	}
}
