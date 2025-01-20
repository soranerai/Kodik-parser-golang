package video_utils

import (
	"fmt"
	"io"
	"kodik_parser/utils"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
)

const chunkSize = 5 * 1024 * 1024 // Размер части - 5MB

func DownloadVideos(result utils.HandleResult, config *utils.Config) utils.HandleResult {
	var wg sync.WaitGroup
	downloadResults := make(chan utils.Result, len(result.Results))

	semaphore := make(chan struct{}, config.MaxVideosDownloads)
	defer close(semaphore)

	bar := progressbar.DefaultBytes(
		-1,
		"Загрузка видео...",
	)

	for _, res := range result.Results {
		semaphore <- struct{}{} // Захват семафора
		wg.Add(1)
		go func(res utils.Result) {
			defer wg.Done()
			defer func() { <-semaphore }()
			if err := downloadVideo(res, downloadResults, bar, config, result.TitleName); err != nil {
				log.Printf("Failed to download video %s: %v", res.Seria.Num, err)
			}
		}(res)
	}

	go func() {
		wg.Wait()
		close(downloadResults)
	}()

	var downloadResultsArr []utils.Result
	for res := range downloadResults {
		downloadResultsArr = append(downloadResultsArr, res)
	}

	result.Results = utils.SortResults(downloadResultsArr)
	return result
}

func downloadVideo(result utils.Result, downloadResults chan<- utils.Result, bar *progressbar.ProgressBar, config *utils.Config, titleName string) error {
	url := strings.Replace(result.Video, ":hls:manifest.m3u8", "", -1)

	headResp, err := http.Head(url)
	if err != nil {
		return fmt.Errorf("failed to send HEAD request: %w", err)
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch video: status code %d", headResp.StatusCode)
	}

	totalSize := headResp.ContentLength

	numChunks := int(totalSize / chunkSize)
	if totalSize%chunkSize != 0 {
		numChunks++
	}

	tempFiles := make([]string, numChunks)
	var chunkWG sync.WaitGroup

	path := getPath(titleName)

	semaphore := make(chan struct{}, config.MaxVideoWorkers)

	for i := 0; i < numChunks; i++ {
		semaphore <- struct{}{}
		chunkWG.Add(1)
		go func(i int) {
			defer chunkWG.Done()
			defer func() { <-semaphore }()
			start := int64(i) * chunkSize
			end := start + chunkSize - 1
			if end >= totalSize {
				end = totalSize - 1
			}

			tempFile := fmt.Sprintf("%s\\%s_chunk_%d.tmp", path, result.Seria.Num, i)
			if err := downloadChunk(url, start, end, tempFile, bar); err != nil {
				log.Printf("Failed to download chunk %d: %v", i, err)
			}
			tempFiles[i] = tempFile
		}(i)
	}

	chunkWG.Wait()

	outputFile := fmt.Sprintf("%s\\%s_серия.mp4", path, result.Seria.Num)
	if err := mergeChunks(tempFiles, outputFile); err != nil {
		return fmt.Errorf("failed to merge chunks: %w", err)
	}

	downloadResults <- utils.Result{
		Seria: result.Seria,
		Path:  outputFile,
	}
	return nil
}

func downloadChunk(url string, start, end int64, tempFile string, bar *progressbar.ProgressBar) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	req.Header.Set("Range", rangeHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	progressReader := io.TeeReader(resp.Body, bar)

	// Copy from the progressReader to the file
	_, err = io.Copy(file, progressReader)
	if err != nil {
		return fmt.Errorf("failed to write chunk to file: %w", err)
	}

	return nil
}

func mergeChunks(tempFiles []string, outputFile string) error {
	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	for _, tempFile := range tempFiles {
		file, err := os.Open(tempFile)
		if err != nil {
			return fmt.Errorf("failed to open temp file: %w", err)
		}
		_, err = io.Copy(out, file)
		file.Close()
		if err != nil {
			return fmt.Errorf("failed to append chunk: %w", err)
		}
		os.Remove(tempFile)
	}

	return nil
}

func getPath(titleName string) string {
	filePath := fmt.Sprintf("videos\\%s", titleName)
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("Ошибка при получении пути: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		err = os.MkdirAll(absPath, os.ModePerm)
		if err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}
	}

	return absPath
}
