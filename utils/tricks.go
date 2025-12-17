package utils

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

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
	log.Printf(" Normalizing URL: %s", input)

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

// Проверка, что строка состоит только из допустимых Base64-символов.
func isValidBase64Format(s string) bool {
	// Base64 допускает A-Za-z0-9+/, а также может заканчиваться на = или ==
	base64Regex := regexp.MustCompile(`^[A-Za-z0-9+/]+={0,2}$`)
	return base64Regex.MatchString(s)
}

// decodeROT применяет обратную ротацию для букв латинского алфавита с указанным сдвигом.
func decodeROT(s string, shift int) string {
	shift = shift % 26
	var result strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			newRune := r - rune(shift)
			if newRune < 'A' {
				newRune += 26
			}
			result.WriteRune(newRune)
		case r >= 'a' && r <= 'z':
			newRune := r - rune(shift)
			if newRune < 'a' {
				newRune += 26
			}
			result.WriteRune(newRune)
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// base64Decode пытается декодировать строку как Base64.
func base64Decode(s string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}

// Вычисляет «оценку» расшифрованной строки.
// Оценка методом эвристики. Может меняться по необходимости
func scoreCandidate(decoded string) int {
	score := 0

	// строка без запретных символов - самый ценный параметр
	if isCleanString(decoded) {
		score += 50
	}
	// нужный нам домен
	if strings.Contains(decoded, "kodik-storage.com") {
		score += 20
	}
	if strings.Contains(decoded, "://") {
		score += 20
	}
	if strings.HasPrefix(decoded, "//") {
		score += 10
	}
	if strings.Contains(decoded, ".mp4") {
		score += 5
	}
	if strings.Contains(decoded, ".m3u8") {
		score += 5
	}
	return score
}

// проверяет, что строка не содержит запрещенных в URL символов
func isCleanString(s string) bool {
	allowedSymbols := ":/.?&=%-_#[]"
	for _, r := range s {
		if !unicode.IsPrint(r) && !strings.ContainsRune(allowedSymbols, r) {
			return false
		}
		if r == '\ufffd' {
			return false
		}
	}
	return true
}

// Реверс строки
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// пытается расшифровать строку путем перебора всех вариантов, что я видел у Kodik
func AutoDecode(input string) (string, error) {
	log.Printf("Decoding string: %s", input)
	type decodingResult struct {
		decoded string
		score   int
	}

	var channel = make(chan decodingResult, 3)
	var wg sync.WaitGroup
	wg.Add(3)

	// Пробуем вариант base64
	go func() {
		if isValidBase64Format(input) {
			candidate, err := base64Decode(input)
			score := scoreCandidate(candidate)
			if err == nil && score > 1 {
				channel <- decodingResult{
					decoded: candidate,
					score:   score,
				}
			}
		}
		wg.Done()
	}()

	// Пробуем вариант reversed base64
	go func() {
		reversed := reverseString(input)
		if isValidBase64Format(reversed) {
			candidate, err := base64Decode(reversed)
			score := scoreCandidate(candidate)
			if err == nil && score > 1 {
				channel <- decodingResult{
					decoded: candidate,
					score:   score,
				}
			}
		}
		wg.Done()
	}()

	// Пробуем вариант ROT + base64
	go func() {
		bestScore := -1
		var bestDecoded string

		for shift := range 26 {
			candidate := decodeROT(input, shift)
			if !isValidBase64Format(candidate) {
				continue
			}

			decoded, err := base64Decode(candidate)
			if err != nil {
				continue
			}

			score := scoreCandidate(decoded)
			if score > bestScore {
				bestScore = score
				bestDecoded = decoded
			}
		}

		if bestScore > 1 {
			channel <- decodingResult{
				decoded: bestDecoded,
				score:   bestScore,
			}
		}

		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(channel)
	}()

	var decodingResults []decodingResult
	for result := range channel {
		decodingResults = append(decodingResults, result)
	}

	if len(decodingResults) > 0 {
		var bestDecodedString string
		bestScore := -1

		for i := range decodingResults {
			if bestScore < decodingResults[i].score {
				bestDecodedString = decodingResults[i].decoded
				bestScore = decodingResults[i].score
			}
		}

		log.Printf("Decoding result: %s", bestDecodedString)
		return bestDecodedString, nil
	}

	return "", errors.New("decoding failure")
}
