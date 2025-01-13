package utils

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
)

// type kodik_domain struct {
// 	Domain      string
// 	Domain_sign string
// }

func GetKodikTest() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://kodik.biz/serial/42283/7fa89e31e53326e9d8ca270ce9fd2046/720p?uid=nymqLn", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	set_headers("", "kodik.biz", req)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}

	defer resp.Body.Close()

	// Проверяем заголовок Content-Encoding
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			fmt.Println("Error creating gzip reader:", err)
			return
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// Создаём файл для записи
	file, err := os.Create("response.html")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Копируем тело ответа в файл
	_, err = io.Copy(file, reader)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("Response saved to response.html")
}

func set_headers(referer string, host string, req *http.Request) {
	req.Header.Set("Host", host)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,ru-RU;q=0.8,ru;q=0.7")

	if false {
		req.Header.Set("Referer", referer)
	}
}
