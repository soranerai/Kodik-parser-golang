package video_utils

import (
	"fmt"
	"io"
	"kodik_parser/utils"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

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

	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	defer client.CloseIdleConnections()

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
			attempts := 3

			for attempts > 0 {
				if err := downloadChunk(client, url, start, end, tempFile, bar); err != nil {
					if strings.Contains(err.Error(), "TLS handshake timeout") || strings.Contains(err.Error(), "status code: 504") {
						log.Printf("Retrying chunk %d due to error: %v", i, err)
						time.Sleep(5 * time.Second)
						attempts--
						continue
					}
					log.Printf("Failed to download chunk %d: %v", i, err)
					return
				}
				tempFiles[i] = tempFile
				break
			}
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

func downloadChunk(client *http.Client, url string, start, end int64, tempFile string, bar *progressbar.ProgressBar) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	req.Header.Set("Range", rangeHeader)

	resp, err := client.Do(req)
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
	filePath := fmt.Sprintf("videos\\%s", normalizeDirName(titleName))
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

func normalizeDirName(dirName string) string {
	// Список зарезервированных имен файлов/папок в Windows
	reservedNames := map[string]struct{}{
		"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
		"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {},
		"COM8": {}, "COM9": {}, "LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {},
		"LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
	}

	// Удаляем или заменяем символы, не разрешенные в Windows
	re := regexp.MustCompile(`[<>:"/\|?*\ ]`)
	dirName = re.ReplaceAllString(dirName, "_")

	// Заменяем зарезервированные имена на "_"
	if _, exists := reservedNames[strings.ToUpper(dirName)]; exists {
		dirName = "_" + dirName
	}

	// Обрезаем имя, если оно слишком длинное (Windows ограничивает длину пути 255 символами)
	if len(dirName) > 255 {
		dirName = dirName[:255]
	}

	// Удаляем пробелы в начале и в конце строки
	dirName = strings.TrimSpace(dirName)

	// Убираем все неалфавитные символы в начале и в конце
	dirName = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			return r
		}
		return -1
	}, dirName)

	return dirName
}
