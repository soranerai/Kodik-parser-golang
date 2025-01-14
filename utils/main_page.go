package utils

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

// GetKodikPage sends a GET request to the specified URL using the provided HTTP client,
// sets the necessary headers, and returns the response body as a string.
// If an error occurs during the request creation, execution, or response reading,
// it prints the error and returns an empty string.
//
// Parameters:
//   - client: *http.Client - The HTTP client to use for sending the request.
//   - url: string - The URL to send the GET request to.
//
// Returns:
//   - string: The response body as a string, or an empty string if an error occurs.
func GetKodikPage(client *http.Client, url string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return ""
	}

	Set_headers("", "kodik.online", false, req)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return ""
	}

	defer resp.Body.Close()

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			fmt.Println("Error creating gzip reader:", err)
			return ""
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println("Error while reading response body:", err)
		return ""
	}

	return string(body)
}

// set_headers sets the necessary HTTP headers for a request.
//
// Parameters:
//   - referer: The referer URL to be set in the "Referer" header if referer_required is true.
//   - host: The host to be set in the "Host" header.
//   - referer_required: A boolean indicating whether the "Referer" header should be set.
//   - req: A pointer to the http.Request object where the headers will be set.
//
// The function sets the following headers:
//   - Host
//   - Connection
//   - Cache-Control
//   - Upgrade-Insecure-Requests
//   - User-Agent
//   - Accept
//   - Accept-Encoding
//   - Accept-Language
//   - Referer (conditionally based on referer_required)
