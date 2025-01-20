package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	KodikLinkTypes = NewKodikLinkTypes()
)

// Структуры данных для параметров и видеоинформации
type KodikParam struct {
	Domain     string
	DomainSign string
}

type KodikParams struct {
	MainDomain    KodikParam
	PlayerDomain  KodikParam
	RefererDomain KodikParam
}

type KodikSerialDetails struct {
	SerialID         string
	SerialHash       string
	PlayerDomain     string
	TranslationID    string
	TranslationTitle string
}

type KodikSeriaInfo struct {
	Num   string
	Id    string
	Hash  string
	Title string
}

type kodikLinkTypes struct {
	Serial int
	Movie  int
}

type Config struct {
	OpenInMpvNet       bool
	MpvNetExecutable   string
	DownloadResults    bool
	MaxVideosDownloads int
	MaxVideoWorkers    int
}

type Result struct {
	Seria KodikSeriaInfo
	Video string
	Path  string
}

func NewKodikLinkTypes() kodikLinkTypes {
	return kodikLinkTypes{
		Serial: 0,
		Movie:  1,
	}
}

// извлекает информацию о сериях из тела страницы плеера
func ParseSeasonSeries(body string) ([]KodikSeriaInfo, error) {
	var seasonInfo []KodikSeriaInfo

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return seasonInfo, err
	}

	var seriaInfo KodikSeriaInfo
	doc.Find(".serial-series-box select option").Each(
		func(i int, s *goquery.Selection) {
			seriaInfo = KodikSeriaInfo{}

			seriaInfo.Num, _ = s.Attr("value")
			seriaInfo.Id, _ = s.Attr("data-id")
			seriaInfo.Hash, _ = s.Attr("data-hash")
			seriaInfo.Title, _ = s.Attr("data-title")

			seasonInfo = append(seasonInfo, seriaInfo)
		})

	return seasonInfo, nil
}

// ParseURLParameters парсит параметры из строки body в структуру KodikParams
func ParseURLParameters(body string, params *KodikParams) error {
	r := regexp.MustCompile(`\{[^{}]*\}`)
	paramsJSON := r.FindString(body)
	if paramsJSON == "" {
		return errors.New("failed to parse params: regex returned empty string")
	}

	var paramsMap map[string]interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &paramsMap); err != nil {
		return errors.New("failed to unmarshal params JSON: " + err.Error())
	}

	params.MainDomain.Domain = getStringValue(paramsMap, "d")
	params.MainDomain.DomainSign = getStringValue(paramsMap, "d_sign")
	params.PlayerDomain.Domain = getStringValue(paramsMap, "pd")
	params.PlayerDomain.DomainSign = getStringValue(paramsMap, "pd_sign")
	params.RefererDomain.Domain = getStringValue(paramsMap, "ref")
	params.RefererDomain.DomainSign = getStringValue(paramsMap, "ref_sign")

	return nil
}

// ParseSerialDetails извлекает детали сериала из строки body
func ParseSerialDetails(body string) (KodikSerialDetails, error) {
	var details KodikSerialDetails

	var err error
	details.SerialID, err = extractRegex(body, `var serialId = Number\((\d+)\)`, "SerialID")
	if err != nil {
		return details, err
	}

	details.SerialHash, err = extractRegex(body, `var serialHash = "([0-9a-z]+)"`, "SerialHash")
	if err != nil {
		return details, err
	}

	details.PlayerDomain, err = extractRegex(body, `var playerDomain = "([a-z.]+)"`, "PlayerDomain")
	if err != nil {
		return details, err
	}

	details.TranslationID, err = extractRegex(body, `var translationId = (\d+)`, "TranslationID")
	if err != nil {
		return details, err
	}

	details.TranslationTitle, err = extractRegex(body, `var translationTitle = "([^"]+)"`, "TranslationTitle")
	if err != nil {
		return details, err
	}

	return details, nil
}

func ParseVideoInfo(body string) ([]KodikSeriaInfo, error) {
	var videoInfo []KodikSeriaInfo

	videoInfo = append(videoInfo, KodikSeriaInfo{})

	var err error
	videoInfo[0].Id, err = extractRegex(body, `videoInfo\.id = \'(\d+)\';`, "videoInfo.Id")
	if err != nil {
		return videoInfo, err
	}

	videoInfo[0].Hash, err = extractRegex(body, `videoInfo\.hash = \'([a-z0-9]+)\';`, "videoInfo.Hash")
	if err != nil {
		return videoInfo, err
	}

	return videoInfo, nil
}

// extractRegex извлекает первую группу по заданной регулярке
func extractRegex(body, pattern, fieldName string) (string, error) {
	r := regexp.MustCompile(pattern)
	match := r.FindStringSubmatch(body)
	if len(match) > 1 {
		return match[1], nil
	}
	return "", errors.New("failed to extract " + fieldName + " using regex")
}

// getStringValue безопасно извлекает строковое значение из карты
func getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

// ParseIframeURL извлекает URL iframe из строки body
func ParseIframeURL(body string) (string, error) {
	url, err := extractRegex(body, `iframe src="([^"]+)"`, "IframeURL")
	if err != nil {
		return "", err
	}
	return "https:" + url, nil
}

// ParseDomainFromURL извлекает домен из URL
func ParseDomainFromURL(url string) (string, error) {
	r := regexp.MustCompile(`https?://([^/]+)`)
	match := r.FindStringSubmatch(url)
	if len(match) > 1 {
		return match[1], nil
	}
	return "", errors.New("failed to parse domain from URL")
}

// GetSerialScriptURL возвращает полный URL для скрипта сериала
func GetSerialScriptURL(body, playerDomain string) (string, error) {
	path, err := extractRegex(body, `<script .+ src="(.+)"></script>`, "ScriptPath")
	if err != nil {
		return "", err
	}
	return "https://" + playerDomain + path, nil
}

// GetSecretMethod извлекает и декодирует секретный метод
func GetSecretMethod(body string) (string, error) {
	encoded, err := extractRegex(body, `atob\("([^"]+)"\)`, "SecretMethod")
	if err != nil {
		return "", err
	}
	return DecodeBase64(encoded)
}

func parseJSONToMap(jsonStr string) (map[string]interface{}, error) {
	// Создаем переменную для хранения результата
	var result map[string]interface{}

	// Парсим JSON в map
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return result, nil
}

func GetBestQualityURL(body string) (string, error) {
	var (
		bestQuality       string
		currentQualityInt int
		bestQualityInt    int
	)

	secretMap, _ := parseJSONToMap(body)

	links, ok := secretMap["links"].(map[string]interface{})
	if !ok {
		return "", errors.New("failed to assert links to map[string]interface{}")
	}

	for currentQuality := range links {
		currentQualityInt, _ = strconv.Atoi(currentQuality)
		bestQualityInt, _ = strconv.Atoi(bestQuality)

		if bestQuality == "" || currentQualityInt > bestQualityInt {
			bestQuality = currentQuality
		}
	}

	resolutions, ok := links[bestQuality].([]interface{})
	if !ok {
		return "", errors.New("failed to assert resolutions to []interface{}")
	}

	resolution, ok := resolutions[0].(map[string]interface{})
	if !ok {
		return "", errors.New("failed to assert resolution to map[string]interface{}")
	}

	decodedURL, err := DecodeVideoUrl(resolution["src"].(string))
	if err != nil {
		return "", fmt.Errorf("ошибка декодирования секретного метода: %w", err)
	}

	return NormalizeURL(decodedURL), nil
}

func GetLinkType(url string) int {
	if strings.Contains(url, "serial") {
		return KodikLinkTypes.Serial
	} else {
		return KodikLinkTypes.Movie
	}
}

func GetConfigFile(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return Config{}, err
	}

	result, _ := parseJSONToMap(string(data))

	config := Config{
		OpenInMpvNet:       result["openInMpvNet"].(bool),
		MpvNetExecutable:   result["mpvNetExecutable"].(string),
		DownloadResults:    result["downloadResults"].(bool),
		MaxVideosDownloads: int(result["maxVideosDownloads"].(float64)),
		MaxVideoWorkers:    int(result["maxVideoWorkers"].(float64)),
	}

	return config, nil
}

func SortResults(results []Result) []Result {
	var (
		sortedResults []Result
		sNumFirst     int
		sNumSecond    int
	)

	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			sNumFirst, _ = strconv.Atoi(results[i].Seria.Num)
			sNumSecond, _ = strconv.Atoi(results[j].Seria.Num)

			if sNumFirst > sNumSecond {
				results[i], results[j] = results[j], results[i]
			}
		}
		sortedResults = append(sortedResults, results[i])
	}
	return sortedResults
}

func ValidateURL(url string) bool {
	// Регулярное выражение для проверки URL
	re := `^https://kodik\.online/(movie|serial)/\d+/[a-zA-Z0-9]+$`

	// Проверка соответствия регулярному выражению
	matched, err := regexp.MatchString(re, url)
	if err != nil {
		fmt.Println("Ошибка при проверке регулярного выражения:", err)
		return false
	}

	return matched
}
