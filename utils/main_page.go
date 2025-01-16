package utils

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

var (
	KodikPage = NewKodikPageType()
)

type kodikPageType struct {
	MAIN_PAGE         int
	PLAYER_PAGE       int
	SERIAL_PAGE       int
	APP_SERIAL_SCRIPT int
	APP_PLAYER_SCRIPT int
}

func NewKodikPageType() kodikPageType {
	return kodikPageType{
		MAIN_PAGE:         0,
		PLAYER_PAGE:       1,
		SERIAL_PAGE:       2,
		APP_SERIAL_SCRIPT: 3, // Если это актуально, добавьте значения для остальных страниц
		APP_PLAYER_SCRIPT: 4,
	}
}

func GetPage(client *http.Client, url string, pageType int, params *KodikParams, forceRef string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Установка заголовков в зависимости от типа страницы
	if err := setHeadersBasedOnPageType(pageType, req, params, forceRef); err != nil {
		return "", err
	}

	// Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Обработка ответа
	body, err := processResponseBody(resp)
	if err != nil {
		return "", err
	}

	return body, nil
}

func setHeadersBasedOnPageType(pageType int, req *http.Request, params *KodikParams, forceRef string) error {
	switch pageType {
	case KodikPage.MAIN_PAGE:
		SetHeaders("", "", req, KodikPage.MAIN_PAGE)
	case KodikPage.PLAYER_PAGE:
		SetHeaders("https://"+params.MainDomain.Domain+"/", "", req, KodikPage.PLAYER_PAGE)
	case KodikPage.APP_SERIAL_SCRIPT:
		SetHeaders(forceRef, "", req, KodikPage.APP_SERIAL_SCRIPT)
	default:
		return fmt.Errorf("unknown page type: %d", pageType)
	}
	return nil
}

func processResponseBody(resp *http.Response) (string, error) {
	var reader io.ReadCloser
	var err error

	// Обрабатываем сжато или не сжато содержимое
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("error creating gzip reader: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("error while reading response body: %w", err)
	}

	return string(body), nil
}
