package main

import (
	"os"
)

func main() {
	if err := Bootstrap(); err != nil {
		os.Exit(1)
	}
}
