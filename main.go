package main

import (
	"errors"
	"fmt"
	"kodik_parser/utils"
	"kodik_parser/video_utils"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

func handleSerial(url string, urlType int) utils.HandleResult {
	var (
		requestParams utils.KodikRequestParams
		responseBody  string
		err           error
		bar           *progressbar.ProgressBar
		handleResult  utils.HandleResult
	)

	fmt.Println("Парсинг сериала...")

	log.Println(" Parsing main page")

	bar = progressbar.Default(5)

	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}
	defer client.CloseIdleConnections()

	// Инициализация параметров
	params := utils.KodikParams{}

	// Получаем домен
	domain, err := utils.ParseDomainFromURL(url)
	if err != nil {
		log.Fatalf("Error parsing domain from URL: %v", err)
	}
	params.MainDomain.Domain = domain

	// Получаем страницу
	requestParams = utils.GetKodikRequestParams(
		url, "", "", "", "", utils.KodikPage.MAIN_PAGE, utils.KodikSeriaInfo{})

	responseBody, err = utils.GetPage(client, &params, requestParams)
	if err != nil {
		log.Fatalf("Error getting page: %v", err)
	}

	bar.Add(1)

	// Извлекаем URL iframe
	playerPageURL, err := utils.ParseIframeURL(responseBody)
	if err != nil {
		log.Fatalf("Error parsing iframe URL: %v", err)
	}

	titleName, err := utils.ParseTitle(responseBody)
	if err != nil {
		log.Fatalf("Error parsing title: %v", err)
	}
	handleResult.TitleName = titleName

	bar.Add(1)

	log.Println(" Parsing player page")

	// Получаем страницу плеера
	requestParams = utils.GetKodikRequestParams(
		playerPageURL, params.MainDomain.Domain, "", "", "", utils.KodikPage.PLAYER_PAGE, utils.KodikSeriaInfo{})

	responseBody, err = utils.GetPage(client, &params, requestParams)
	if err != nil {
		log.Fatalf("Error getting player page: %v", err)
	}

	bar.Add(1)

	log.Println(" Using stealing method 1")

	// Парсим параметры из страницы плеера
	err = utils.ParseURLParameters(responseBody, &params)
	if err != nil {
		log.Fatalf("Error parsing URL parameters: %v", err)
	}

	bar.Add(1)

	var series []utils.KodikSeriaInfo
	if urlType == utils.KodikLinkTypes.Serial {
		// Извлекаем серии сезона
		series, err = utils.ParseSeasonSeries(responseBody)
		if err != nil {
			log.Fatalf("Error parsing series: %v", err)
		}
	} else {
		series, err = utils.ParseVideoInfo(responseBody)
		if err != nil {
			log.Fatalf("Error parsing video: %v", err)
		}
	}

	bar.Add(1)
	bar.Finish()

	// Получаем диапазон серий
	var epRange [2]int
	for {
		epRange, err = getEpisodeRange(len(series))
		if err != nil {
			fmt.Println(err)
		} else {
			break
		}
	}

	fmt.Println("Обход защиты...")

	log.Println(" Serial script manipulations...")

	bar = progressbar.Default(3)

	// Получаем URL для скрипта сериала
	appSerialScriptURL, err := utils.GetSerialScriptURL(responseBody, params.PlayerDomain.Domain)
	if err != nil {
		log.Fatalf("Error getting serial script URL: %v", err)
	}

	bar.Add(1)

	// Получаем скрипт сериала
	requestParams = utils.GetKodikRequestParams(
		appSerialScriptURL, playerPageURL, "", "", "", utils.KodikPage.APP_SERIAL_SCRIPT, utils.KodikSeriaInfo{})

	responseBody, err = utils.GetPage(client, &params, requestParams)
	if err != nil {
		log.Fatalf("Error getting app serial script: %v", err)
	}

	bar.Add(1)

	// Извлекаем расшифрованный секретный метод
	secretMethod, err := utils.GetSecretMethod(responseBody)
	if err != nil {
		log.Fatalf("Error decoding secret method: %v", err)
	}

	bar.Add(1)

	fmt.Println("Получение ссылок...")
	log.Println(" Obtaining secret data")

	bar.Finish()

	bar = progressbar.Default(-1 * int64(epRange[1]-epRange[0]-1))

	// Отправляем POST запросы на секретный метод горутин
	var wg sync.WaitGroup
	results := make(chan utils.Result, 10)
	for i := epRange[0] - 1; i < epRange[1]; i++ {
		wg.Add(1)
		go getVideoUrlWorker(
			&params,
			series[i],
			client, playerPageURL,
			secretMethod, results,
			&wg,
			bar,
			urlType,
		)
	}

	bar.Finish()

	go func() {
		wg.Wait()
		close(results)
	}()

	for Result := range results {
		handleResult.Results = append(handleResult.Results, Result)
	}

	// Сортируем результаты
	handleResult.Results = utils.SortResults(handleResult.Results)

	return handleResult
}

func getVideoUrlWorker(
	params *utils.KodikParams,
	seria utils.KodikSeriaInfo,
	client *http.Client,
	playerPageURL string,
	secretMethod string,
	results chan<- utils.Result,
	wg *sync.WaitGroup,
	bar *progressbar.ProgressBar,
	urlType int) {

	defer wg.Done()

	var (
		requestParams utils.KodikRequestParams
		responseBody  string
		err           error
		video         string
	)

	requestParams = utils.GetKodikRequestParams(
		params.PlayerDomain.Domain+secretMethod,
		playerPageURL,
		params.PlayerDomain.Domain,
		"application/x-www-form-urlencoded; charset=UTF-8",
		"",
		utils.KodikPage.SECRET_METHOD,
		seria,
	)

	responseBody, err = utils.PostPage(client, params, requestParams, urlType)
	if err != nil {
		log.Fatalf("Error getting secret method: %v", err)
	}

	video, err = utils.GetBestQualityURL(responseBody)
	if err != nil {
		log.Fatalf("Error getting best quality URL: %v", err)
	}

	results <- utils.Result{Seria: seria, Video: video}
	bar.Add(1)
}

func handle(url string, config *utils.Config) {
	log.Println("=================LETS FUCK KODIK=================")

	var (
		result utils.HandleResult
	)

	urlType := utils.GetLinkType(url)

	switch urlType {
	case utils.KodikLinkTypes.Serial, utils.KodikLinkTypes.Movie:
		result = handleSerial(url, urlType)
	}

	if config.DownloadResults {
		fmt.Println("Загрузка видео...")
		log.Print(" Video download is starting")
		result = video_utils.DownloadVideos(result, config)
		log.Print(" Video download is complete")
	}

	if config.OpenInMpvNet {
		log.Print(" Opening in MPV")
		utils.OpenInMpvNet(result, config)
	} else {
		log.Print(" Printing results")
		utils.PrintResults(result)
	}

	log.Println("============KODIK SUCCESSFULLY FUCKED============")
}

func getEpisodeRange(epCount int) ([2]int, error) {
	if epCount == 1 {
		return [2]int{1, 1}, nil
	}

	var input string
	var result [2]int

	fmt.Printf("Введите диапазон серий (от 1 до %d) через дефис: ", epCount)
	fmt.Scanln(&input)

	parts := strings.Split(input, "-")
	if len(parts) == 2 {
		start, _ := strconv.Atoi(parts[0])
		end, _ := strconv.Atoi(parts[1])
		result = [2]int{start, end}
	} else {
		fmt.Println("Invalid input")
	}

	if result[0] > result[1] {
		result[0], result[1] = result[1], result[0]
	}

	if result[0] < 1 || result[1] < 1 || result[1] > epCount {
		return [2]int{}, errors.New("неверный диапазон")
	}

	return result, nil
}

func main() {
	var (
		url string
	)

	err := utils.InitLogger()
	if err != nil {
		fmt.Println(err)
	}

	config, err := utils.GetConfigFile("config.json")
	if err != nil {
		log.Fatalf("Error getting config file: %v", err)
	}

	for {
		fmt.Print("Введите URL: ")
		fmt.Scanln(&url)

		if !utils.ValidateURL(url) {
			fmt.Println("Некорретный URL!")
			log.Println("Invalid url input")
			continue
		} else {
			break
		}
	}

	url = utils.NormalizeURL(url)

	handle(url, &config)
}
