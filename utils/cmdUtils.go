package utils

import (
	"fmt"
	"log"
	"os/exec"
)

func OpenInMpvNet(results []Result, config *Config) error {
	var commands []string

	commands = append(commands, "append")

	if config.DownloadResults {
		for _, res := range results {
			commands = append(commands, res.Path)
		}
	} else {
		for _, res := range results {
			commands = append(commands, res.Video)
		}
	}

	cmd := exec.Command(config.MpvNetExecutable, commands...)
	err := cmd.Start()
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
