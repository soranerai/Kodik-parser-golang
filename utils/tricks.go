package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// type CustomHeader struct {
// 	name  string
// 	value string
// }

// SetHeaders устанавливает необходимые заголовки в зависимости от типа страницы.
func SetHeaders(req *http.Request, kodikPageType int, params *KodikParams, requestParams KodikRequestParams) error {
	switch kodikPageType {
	case KodikPage.MAIN_PAGE, KodikPage.SERIAL_PAGE:
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Cache-Control", "max-age=0")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,ru-RU;q=0.8,ru;q=0.7")
	case KodikPage.SECRET_METHOD:
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("Accept-encoding", "gzip, deflate, br, zstd")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Content-Type", requestParams.content_type) //"application/x-www-form-urlencoded; charset=UTF-8"
		req.Header.Set("Origin", "https://"+params.PlayerDomain.Domain)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36")

	if requestParams.host != "" {
		req.Header.Set("Host", requestParams.host)
	}

	if requestParams.referer != "" {
		req.Header.Set("Referer", requestParams.referer)
	}

	return nil
}

func GetSecretMethodPayload(params *KodikParams, seria KodikSeriaInfo, urlType int) *bytes.Buffer {
	payload := url.Values{}

	payload.Set("d", params.MainDomain.Domain)
	payload.Set("d_sign", params.MainDomain.DomainSign)

	payload.Set("pd", params.PlayerDomain.Domain)
	payload.Set("pd_sign", params.PlayerDomain.DomainSign)

	payload.Set("ref", NormalizeURL(params.RefererDomain.Domain)) //params.RefererDomain.Domain
	payload.Set("ref_sign", params.RefererDomain.DomainSign)

	payload.Set("bad_user", "false")
	payload.Set("cdn_is_working", "true")
	payload.Set("uid", "numqLn")

	if urlType == KodikLinkTypes.Serial {
		payload.Set("type", "seria")
	} else {
		payload.Set("type", "video")
	}

	payload.Set("hash", seria.Hash)
	payload.Set("id", seria.Id)
	payload.Set("info", "{}")

	return bytes.NewBufferString(payload.Encode())
}

// rot13 выполняет преобразование строки с использованием алгоритма ROT13.
func rot13(input string) string {
	var result strings.Builder
	for _, char := range input {
		switch {
		case 'A' <= char && char <= 'Z':
			result.WriteRune('A' + (char-'A'+13)%26)
		case 'a' <= char && char <= 'z':
			result.WriteRune('a' + (char-'a'+13)%26)
		default:
			result.WriteRune(char)
		}
	}
	return result.String()
}

// декодирует строку, закодированную в base64.
func DecodeBase64(encoded string) (string, error) {
	if encoded[0] == '=' {
		encoded = ReverseString(encoded)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("error while decoding base64: %w", err)
	}
	return string(decoded), nil
}

// выполняет декодирование секрета методом ROT13, а затем base64.
func DecodeVideoUrl(secretMethod string) (string, error) {
	rot13Src := rot13(secretMethod)
	decoded, err := DecodeBase64(rot13Src)
	if err != nil {
		return "", fmt.Errorf("error decoding secret method: %w", err)
	}

	return decoded, nil
}

// переворачивает строку
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Костыльная функция, использует normalizeURL для нормализации URL, но возвращает пустую строку, если входная строка пустая
func NormalizeURL(input string) string {
	if input == "" {
		return ""
	}

	res, err := normalizeURL(input)
	if err != nil {
		log.Fatal(err)
		return ""
	}

	return res
}

// нормализует URL, добавляя схему и завершающий слеш
func normalizeURL(input string) (string, error) {
	input, err := url.QueryUnescape(input)
	if err != nil {
		return "", fmt.Errorf("ошибка декодирования URL: %w", err)
	}

	if strings.HasPrefix(input, "//") {
		input = "https:" + input
	} else if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		input = "https://" + input
	}

	// Парсим URL
	parsedURL, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("не удалось распарсить URL: %w", err)
	}

	// Проверяем наличие хоста
	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL должен содержать доменное имя")
	}

	// Добавляем завершающий слеш, если у пути нет ни символов, ни слеша
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	// Возвращаем нормализованный URL
	return parsedURL.String(), nil
}
