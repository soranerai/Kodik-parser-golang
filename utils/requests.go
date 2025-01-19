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
	SECRET_METHOD     int
}

type KodikRequestParams struct {
	url          string
	referer      string
	origin       string
	content_type string
	host         string
	page_type    int
	seria        KodikSeriaInfo
}

func NewKodikPageType() kodikPageType {
	return kodikPageType{
		MAIN_PAGE:         0,
		PLAYER_PAGE:       1,
		SERIAL_PAGE:       2,
		APP_SERIAL_SCRIPT: 3,
		APP_PLAYER_SCRIPT: 4,
		SECRET_METHOD:     5,
	}
}

func newKodikRequestParams(url string, referer string, origin string, content_type string, host string, page_type int, seria KodikSeriaInfo) KodikRequestParams {
	return KodikRequestParams{
		url:          url,
		referer:      referer,
		origin:       origin,
		content_type: content_type,
		host:         host,
		page_type:    page_type,
		seria:        seria,
	}
}

// Возращает параметры запроса для Kodik с нормализованными URL
func GetKodikRequestParams(url string, referer string, origin string, content_type string, host string, page_type int, seria_info KodikSeriaInfo) KodikRequestParams {
	return newKodikRequestParams(
		NormalizeURL(url), NormalizeURL(referer), NormalizeURL(origin), content_type, NormalizeURL(host), page_type, seria_info)
}

func GetPage(client *http.Client, kodikParams *KodikParams, requestParams KodikRequestParams) (string, error) {
	req, err := http.NewRequest("GET", requestParams.url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Установка заголовков
	SetHeaders(req, requestParams.page_type, kodikParams, requestParams)

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

func PostPage(client *http.Client, kodikParams *KodikParams, requestParams KodikRequestParams) (string, error) {
	var (
		req *http.Request
		err error
	)

	switch requestParams.page_type {
	case KodikPage.SECRET_METHOD:
		req, err = http.NewRequest("POST", requestParams.url, GetSecretMethodPayload(kodikParams, requestParams.seria))
	default:
		req, err = http.NewRequest("POST", requestParams.url, nil)
	}

	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Установка заголовков
	SetHeaders(req, requestParams.page_type, kodikParams, requestParams)

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
