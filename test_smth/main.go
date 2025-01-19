package main

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

func main() {
	// Create two progress bars that will be displayed in one stream
	bar1 := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("Downloading File 1"),
		progressbar.OptionSetWriter(os.Stdout), // Простой вывод в консоль
	)

	// Create a new file to write the progress bars
	file, err := os.Create("progress_output.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Create the second progress bar
	bar2 := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("Downloading File 2"),
		progressbar.OptionSetWriter(os.Stdout), // Простой вывод в консоль
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	// Запуск параллельных горутин
	go func() {
		for i := 0; i <= 100; i++ {
			bar1.Add(1)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	go func() {
		for i := 0; i <= 100; i++ {
			bar2.Add(1)
			time.Sleep(60 * time.Millisecond)
		}
	}()

	// Ожидание завершения работы горутин
	time.Sleep(6 * time.Second)

	fmt.Println("\nDownloads complete.")
}
