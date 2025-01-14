package utils

import (
	"encoding/json"
	"regexp"
)

// ` = alt + 96

type kodik_param struct {
	Domain      string
	Domain_sign string
}

type kodik_params struct {
	main_domain    kodik_param
	player_domain  kodik_param
	referer_domain kodik_param
}

type kodik_serial_details struct {
	serialId         string
	serialHash       string
	playerDomain     string
	translationId    string
	translationTitle string
	cdnCheckLink     string
}

func Parse_url_parameters(body string) kodik_params {
	params := kodik_params{}

	r, _ := regexp.Compile(`\{[^{}]*\}`)
	params_json_string := r.FindString(body)

	if params_json_string == "" {
		panic("Error while parsing params from player page (regex gives empty string)")
	}

	var params_map map[string]interface{}
	err := json.Unmarshal([]byte(params_json_string), &params_map)
	if err != nil {
		panic("Error while parsing params from player page (string to map failure)")
	}

	params.main_domain.Domain, _ = params_map["d"].(string)
	params.main_domain.Domain_sign, _ = params_map["d_sign"].(string)
	params.player_domain.Domain, _ = params_map["pd"].(string)
	params.player_domain.Domain_sign, _ = params_map["pd_sign"].(string)
	params.referer_domain.Domain, _ = params_map["ref"].(string)
	params.referer_domain.Domain_sign, _ = params_map["ref_sign"].(string)

	return params
}

// Parse_serial_details extracts serial details from the provided HTML body string.
// It uses regular expressions to find and parse specific variables within the HTML body.
// If any of the required details are not found, the function will panic.
//
// Parameters:
//   - body: A string containing the HTML body from which to parse the serial details.
//
// Returns:
//   - kodik_serial_details: A struct containing the parsed serial details.
//
// Panics:
//   - If any of the required details (serialId, serialHash, playerDomain, translationId, translationTitle)
//     are not found in the provided HTML body, the function will panic with an appropriate error message.
func Parse_serial_details(body string) kodik_serial_details {
	details := kodik_serial_details{}

	details.serialId = extractDetail(body, `var serialId = Number\((\d+)\)`, "(serialId)")
	details.serialHash = extractDetail(body, `var serialHash \= \"([0-9a-z]+)\"`, "(serialHash)")
	details.playerDomain = extractDetail(body, `var playerDomain \= \"([a-z\.]+)\"`, "(playerDomain)")
	details.translationId = extractDetail(body, `var translationId \= (\d+)`, "(translationId)")
	details.translationTitle = extractDetail(body, `var translationTitle \= \"([a-zA-Zа-яА-Я0-9 \.]+)\"`, "(translationTitle)")

	return details
}

func extractDetail(body, pattern, detailName string) string {
	r := regexp.MustCompile(pattern)
	match := r.FindStringSubmatch(body)
	if len(match) > 1 {
		return match[1]
	}
	panic("Error while parsing details from player page " + detailName)
}
