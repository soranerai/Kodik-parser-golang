package utils

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// SetHeaders устанавливает необходимые заголовки в зависимости от типа страницы.
func SetHeaders(referer string, host string, req *http.Request, kodikPageType int) error {
	switch kodikPageType {
	case KodikPage.MAIN_PAGE, KodikPage.SERIAL_PAGE:
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Cache-Control", "max-age=0")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,ru-RU;q=0.8,ru;q=0.7")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36")

	if host != "" {
		req.Header.Set("Host", host)
	}

	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	return nil
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
func DecodeSecretMethod(secretMethod string) (string, error) {
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
