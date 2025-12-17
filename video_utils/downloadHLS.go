package video_utils

import (
	"context"
	"fmt"
	"io"
	"kodik_parser/utils"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

type downloadedHlsFragment struct {
	Number int
	Data   []byte
}

type HlsFragment struct {
	Number   int
	Duration int
	Url      string
}

func DownloadVideosHLS(result utils.HandleResult, config *utils.Config) utils.HandleResult {
	var wg sync.WaitGroup

	// открываем семафор для ограничения количества одновременно загружаемых файлов
	semaphore := make(chan struct{}, config.MaxVideosDownloads)
	defer close(semaphore)

	bar := progressbar.DefaultBytes(
		-1,
		"Загрузка видео...",
	)

	for _, res := range result.Results {
		semaphore <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			if path, err := downloadVideoHls(res, bar, config, result.TitleName); err != nil {
				log.Printf("Failed to download HLS seria %s: %v", res.Seria.Num, err)
			} else {
				res.Path = path
			}
		}()
	}

	wg.Wait()

	return result
}

func getPlaylist(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error getting playlist: %v", err)
	} else if resp.Body == nil {
		return "", fmt.Errorf("empty playlist")
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error converting response body: %v", err)
	}
	body := buf.String()

	return body, nil
}

func parseHlsFragments(body, baseUrl string) ([]HlsFragment, error) {
	var fragments []HlsFragment

	r := regexp.MustCompile(`#EXTINF:(\d+\.\d+),\n(\S+)`)

	fragments_raw := r.FindAllStringSubmatch(body, -1)

	if len(fragments_raw) < 1 {
		return nil, fmt.Errorf("can't find any fragment in playlist")
	}

	for i, fragment := range fragments_raw {
		duration := fragment[1]
		url := fragment[2]

		fragments = append(fragments, HlsFragment{
			Number:   i,
			Duration: int(duration[0] - '0'),
			Url:      baseUrl + url[2:],
		})
	}

	return fragments, nil
}

func downloadHlsFragment(client *http.Client, hlsFragment HlsFragment) (downloadedHlsFragment, error) {
	req, err := http.NewRequest("GET", hlsFragment.Url, nil)
	if err != nil {
		return downloadedHlsFragment{}, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return downloadedHlsFragment{}, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return downloadedHlsFragment{}, fmt.Errorf("unexpected status code: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return downloadedHlsFragment{}, fmt.Errorf("error reading resp.Body: %v", err)
	}

	return downloadedHlsFragment{
		Number: hlsFragment.Number,
		Data:   body,
	}, nil
}

func getBaseUrl(url string) string {
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash == -1 {
		log.Fatal("Invalid url")
	}

	return url[:lastSlash+1]
}

func downloadVideoHls(result utils.Result, bar *progressbar.ProgressBar, config *utils.Config, titleName string) (string, error) {
	videoHlsPlaylistBody, err := getPlaylist(result.Video)
	if err != nil {
		return "", fmt.Errorf("error downloading hls video: %v", err)
	}

	hlsPlaylistFragments, err := parseHlsFragments(videoHlsPlaylistBody, getBaseUrl(result.Video))
	if err != nil {
		return "", fmt.Errorf("error downloading hls video: %v", err)
	}

	client := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	defer client.CloseIdleConnections()

	// Канал для получения результатов из горутин
	downloadedFrags := make(chan downloadedHlsFragment, 20)
	// Контекст для прерывания выполнения горутин
	ctx, cancel := context.WithCancel(context.Background())

	// Открываем семафор для ограничения одновременного количества загружаемых фрагментов
	semaphore := make(chan struct{}, config.MaxVideoWorkers)

	var wgDownloader sync.WaitGroup

	// Загружающая горутина
	go func() {
		defer close(downloadedFrags)
		defer wgDownloader.Wait()

		for _, playlistFragment := range hlsPlaylistFragments {
			wgDownloader.Add(1)
			semaphore <- struct{}{}

			select {
			case <-ctx.Done():
				log.Print("context cancel signal recieved")
				return
			default:
			}

			go func() {
				defer wgDownloader.Done()
				defer func() { <-semaphore }()

				var downloadedFragment downloadedHlsFragment

				attempts := 3
				for attempts > 0 {
					downloadedFragment, err = downloadHlsFragment(client, playlistFragment)
					if err != nil {
						log.Printf("failed to download fragment %d with error: %v", playlistFragment.Number, err)

						if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "504") {
							log.Printf("condition fullfilled, retrying to download fragment %d", playlistFragment.Number)
							attempts--
							continue
						} else {
							// В случае непредвиденной ошибки отменяем контекст загрузчика конкретно этого видео
							cancel()
							return
						}
					}

					downloadedFrags <- downloadedFragment
					break
				}
			}()
		}
	}()

	path := fmt.Sprintf("%s\\%s_серия.ts", getPath(titleName), result.Seria.Num)
	file, err := os.Create(path)
	if err != nil {
		cancel()
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	var wgWriter sync.WaitGroup
	wgWriter.Add(1)

	// Записывающая горутина
	go func() {
		defer wgWriter.Done()

		// Буфер для хранения фрагментов, которые пришли не по порядку
		buffer := make(map[int]downloadedHlsFragment)
		expectedFragmentNumber := 0

		for {
			select {
			case <-ctx.Done():
				log.Print("context cancel signal recieved")
				os.Remove(path)
				return

			case downloadedFragment, ok := <-downloadedFrags:
				if ok {
					buffer[downloadedFragment.Number] = downloadedFragment
				}

				for {
					if fragment, exists := buffer[expectedFragmentNumber]; exists {
						// Записываем фрагмент в файл
						_, err := file.Write(fragment.Data)
						if err != nil {
							log.Printf("failed to write fragment %d: %v", fragment.Number, err)
							cancel()
							return
						}

						bar.Add(len(fragment.Data))

						// Удаляем записанный фрагмент из буфера
						delete(buffer, expectedFragmentNumber)

						expectedFragmentNumber++
					} else if !ok && expectedFragmentNumber < len(hlsPlaylistFragments) {
						expectedFragmentNumber++
					} else {
						// Если фрагмент не ожидаемый, выходим из цикла
						break
					}
				}

				if !ok {
					return
				}
			}
		}
	}()

	wgWriter.Wait()

	return path, nil
}
