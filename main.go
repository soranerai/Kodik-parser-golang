package main

import (
	"fmt"
	"kodik_parser/utils"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
)

func handleSerial(url string) []utils.Result {
	var (
		requestParams utils.KodikRequestParams
		responseBody  string
		err           error
		bar           *progressbar.ProgressBar
	)

	fmt.Println("Парсинг сериала...")

	bar = progressbar.Default(5)

	client := &http.Client{}
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

	bar.Add(1)

	// Получаем страницу плеера
	requestParams = utils.GetKodikRequestParams(
		playerPageURL, params.MainDomain.Domain, "", "", "", utils.KodikPage.PLAYER_PAGE, utils.KodikSeriaInfo{})

	responseBody, err = utils.GetPage(client, &params, requestParams)
	if err != nil {
		log.Fatalf("Error getting player page: %v", err)
	}

	bar.Add(1)

	// Парсим параметры из страницы плеера
	err = utils.ParseURLParameters(responseBody, &params)
	if err != nil {
		log.Fatalf("Error parsing URL parameters: %v", err)
	}

	bar.Add(1)

	// Извлекаем серии сезона
	seasonSeries, err := utils.ParseSeasonSeries(responseBody)
	if err != nil {
		log.Fatalf("Error parsing seasonSeries: %v", err)
	}

	bar.Add(1)

	// Получаем диапазон серий
	epRange := getEpisodeRange(len(seasonSeries.Series))

	fmt.Println("Обход защиты...")

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

	bar = progressbar.Default(int64(epRange[0] - 1 + epRange[1]))

	// Отправляем POST запросы на секретный метод горутин
	var wg sync.WaitGroup
	results := make(chan utils.Result, 10)
	for i := epRange[0] - 1; i < epRange[1]; i++ {
		wg.Add(1)
		go getVideoUrlWorker(
			&params,
			seasonSeries.Series[i],
			client, playerPageURL,
			secretMethod, results,
			&wg,
			bar,
		)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var resultsArr []utils.Result
	for Result := range results {
		resultsArr = append(resultsArr, Result)
	}

	// Сортируем результаты
	resultsArr = sortResults(resultsArr)

	return resultsArr
}

func sortResults(results []utils.Result) []utils.Result {
	var (
		sortedResults []utils.Result
		sNumFirst     int
		sNumSecond    int
	)

	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			sNumFirst, _ = strconv.Atoi(results[i].Seria.Num)
			sNumSecond, _ = strconv.Atoi(results[j].Seria.Num)

			if sNumFirst > sNumSecond {
				results[i], results[j] = results[j], results[i]
			}
		}
		sortedResults = append(sortedResults, results[i])
	}
	return sortedResults
}

func getVideoUrlWorker(params *utils.KodikParams, seria utils.KodikSeriaInfo, client *http.Client, playerPageURL string, secretMethod string, results chan<- utils.Result, wg *sync.WaitGroup, bar *progressbar.ProgressBar) {
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

	responseBody, err = utils.PostPage(client, params, requestParams)
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
	var (
		results []utils.Result
	)

	switch utils.GetLinkType(url) {
	case utils.KodikLinkTypes.Serial:
		results = handleSerial(url)
	}

	if config.OpenInMpvNet {
		utils.OpenInMpvNet(results, config)
	} else {
		utils.PrintResults(results)
	}
}

func getEpisodeRange(epCount int) [2]int {
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
		log.Fatalf("Неверный диапазон серий")
	}

	return result
}

func main() {
	var (
		url string
	)

	config, err := utils.GetConfigFile("config.json")
	if err != nil {
		log.Fatalf("Error getting config file: %v", err)
	}

	fmt.Print("Введите URL: ")
	fmt.Scanln(&url)

	url = utils.NormalizeURL(url)

	handle(url, &config)
}
