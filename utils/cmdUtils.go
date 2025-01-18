package utils

import (
	"fmt"
	"log"
	"os/exec"
)

func OpenInMpvNet(results []Result, config *Config) error {
	var commands []string

	commands = append(commands, "append")

	for _, res := range results {
		commands = append(commands, res.Video)
	}

	cmd := exec.Command(config.MpvNetExecutable, commands...)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error while opening with MVP.NET!")
		return err
	}

	return nil
}

func PrintResults(results []Result) {
	for _, res := range results {
		fmt.Printf("Серия %s: %s\n", res.Seria.Num, res.Video)
	}
}
