package main

import (
	"fmt"
	"kodik_parser/utils"
	"log"
	"net/http"
)

func main() {
	var (
		requestParams utils.KodikRequestParams
		responseBody  string
		err           error
	)

	client := &http.Client{}
	defer client.CloseIdleConnections()

	// Инициализация параметров
	params := utils.KodikParams{}
	url := "https://kodik.online/serial/42283/nymqLn7fa89e31"

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

	// Извлекаем URL iframe
	playerPageURL, err := utils.ParseIframeURL(responseBody)
	if err != nil {
		log.Fatalf("Error parsing iframe URL: %v", err)
	}

	// Получаем страницу плеера
	requestParams = utils.GetKodikRequestParams(
		playerPageURL, params.MainDomain.Domain, "", "", "", utils.KodikPage.PLAYER_PAGE, utils.KodikSeriaInfo{})

	responseBody, err = utils.GetPage(client, &params, requestParams)
	if err != nil {
		log.Fatalf("Error getting player page: %v", err)
	}

	// Парсим параметры из страницы плеера
	err = utils.ParseURLParameters(responseBody, &params)
	if err != nil {
		log.Fatalf("Error parsing URL parameters: %v", err)
	}

	// // Извлекаем детали сериала
	// serialDetails, err := utils.ParseSerialDetails(responseBody)
	// if err != nil {
	// 	log.Fatalf("Error parsing serial details: %v", err)
	// }

	// Извлекаем серии сезона
	seasonSeries, err := utils.ParseSeasonSeries(responseBody)
	if err != nil {
		log.Fatalf("Error parsing seasonSeries: %v", err)
	}

	// Получаем URL для скрипта сериала
	appSerialScriptURL, err := utils.GetSerialScriptURL(responseBody, params.PlayerDomain.Domain)
	if err != nil {
		log.Fatalf("Error getting serial script URL: %v", err)
	}

	// Получаем скрипт сериала
	requestParams = utils.GetKodikRequestParams(
		appSerialScriptURL, playerPageURL, "", "", "", utils.KodikPage.APP_SERIAL_SCRIPT, utils.KodikSeriaInfo{})

	responseBody, err = utils.GetPage(client, &params, requestParams)
	if err != nil {
		log.Fatalf("Error getting app serial script: %v", err)
	}

	// Извлекаем расшифрованный секретный метод
	secretMethod, err := utils.GetSecretMethod(responseBody)
	if err != nil {
		log.Fatalf("Error decoding secret method: %v", err)
	}

	// Отправляем POST запрос на секретный метод

	requestParams = utils.GetKodikRequestParams(
		params.PlayerDomain.Domain+secretMethod,
		playerPageURL,
		params.PlayerDomain.Domain,
		"application/x-www-form-urlencoded; charset=UTF-8",
		"",
		utils.KodikPage.SECRET_METHOD,
		seasonSeries.Series[0],
	)

	responseBody, err = utils.PostPage(client, &params, requestParams)
	if err != nil {
		log.Fatalf("Error getting secret method: %v", err)
	}

	video, err := utils.GetBestQualityURL(responseBody)
	if err != nil {
		log.Fatalf("Error getting best quality URL: %v", err)
	}

	// Печатаем результаты для проверки
	fmt.Println("Serial series:", seasonSeries)
	fmt.Println("Secret Method:", secretMethod)
	fmt.Println("Response body:", responseBody)
	fmt.Println("BestVideoUrl:", video)
}
