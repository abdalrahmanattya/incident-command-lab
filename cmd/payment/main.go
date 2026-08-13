package main

import "github.com/abdalrahmanattya/incident-command-lab/internal/worker"

func main() {
	if err := worker.Run("payment"); err != nil {
		panic(err)
	}
}
