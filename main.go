package main

import (
	"kodik_parser/utils"
	"net/http"
)

func main() {
	client := &http.Client{}

	responseBody := utils.GetKodikPage(client, "https://kodik.online/serial/42283/nymqLn7fa89e31")

}
