package main

import "github.com/abdalrahmanattya/incident-command-lab/internal/server"

func main() {
	if err := server.Run("gateway"); err != nil {
		panic(err)
	}
}
