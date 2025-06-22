/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"os"

	"github.com/richinosan/wpc1-mp3-meta/pkg/cmd"
)

func main() {
	err := cmd.Command.Execute()
	if err != nil {
		os.Exit(1)
	}
}
