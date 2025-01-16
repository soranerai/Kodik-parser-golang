package main

import (
	"fmt"
	"log"
	"net/http"

	"kodik_parser/utils" // Замените на актуальный путь
)

func main() {
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
	responseBody, err := utils.GetPage(client, url, utils.KodikPage.MAIN_PAGE, &params, "")
	if err != nil {
		log.Fatalf("Error getting page: %v", err)
	}

	// Извлекаем URL iframe
	playerPageURL, err := utils.ParseIframeURL(responseBody)
	if err != nil {
		log.Fatalf("Error parsing iframe URL: %v", err)
	}

	// Получаем страницу плеера
	responseBody, err = utils.GetPage(client, playerPageURL, utils.KodikPage.PLAYER_PAGE, &params, "")
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

	// Извлекаем видео информацию
	seasonSeries, err := utils.ParseSeasonSeries(responseBody)
	if err != nil {
		log.Fatalf("Error parsing seasonSeries: %v", err)
	}

	// // Получаем URL для скрипта сериала
	// appSerialURL, err := utils.GetSerialScriptURL(responseBody, params.PlayerDomain.Domain)
	// if err != nil {
	// 	log.Fatalf("Error getting serial script URL: %v", err)
	// }

	// // Получаем скрипт сериала
	// responseBody, err = utils.GetPage(client, appSerialURL, utils.KodikPage.APP_SERIAL_SCRIPT, &params, playerPageURL)
	// if err != nil {
	// 	log.Fatalf("Error getting app serial script: %v", err)
	// }

	// // Извлекаем секретный метод
	// secretMethod, err := utils.GetSecretMethod(responseBody)
	// if err != nil {
	// 	log.Fatalf("Error getting secret method: %v", err)
	// }

	// Печатаем результаты для проверки
	fmt.Println("Serial series:", seasonSeries)
	//fmt.Println("Secret Method:", secretMethod)
}
