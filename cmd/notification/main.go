package main

import "github.com/abdalrahmanattya/incident-command-lab/internal/worker"

func main() {
	if err := worker.Run("notification"); err != nil {
		panic(err)
	}
}
