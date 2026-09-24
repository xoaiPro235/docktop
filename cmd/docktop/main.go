package main

import (
	"fmt"
	"log"

	"github.com/xoaiPro235/docktop/internal/docker"
)

func main() {
	dockercli, err := docker.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer dockercli.Close()

	err = dockercli.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Docker is running")
}
