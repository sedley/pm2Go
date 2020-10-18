package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"gopkg.in/gographics/imagick.v2/imagick"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type AQIResp struct {
	AQI  int
	Desc string
}

type NoPrimaryDataSourceError struct{}

func (e *NoPrimaryDataSourceError) Error() string {
	return "No primary data source found"
}

type APIReading struct {
	IssueDate     string `json:"issueDate"`
	ValidDate     string `json:"ValidDate"`
	IsPrimary     bool   `json:"isPrimary"`
	Aqi           int    `json:"aqi"`
	ReportingArea string `json:"reportingArea"`
	DataType      string `json:"dataType"`
}

type KVPair struct {
	Name  string
	Value string
}

var (
	airnowURL      = "https://airnowgovapi.com/reportingarea/get"
	timeFormat     = "01/02/06"
	requestHeaders = []KVPair{
		{
			Name:  "User-Agent",
			Value: "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:80.0) Gecko/20100101 Firefox/80.0",
		},
		{
			Name:  "Accept",
			Value: "*/*",
		},
		{
			Name:  "Accept-Language",
			Value: "en-US,en;q=0.5",
		},
		{
			Name:  "Content-Type",
			Value: "application/x-www-form-urlencoded; charset=UTF-8",
		},
		{
			Name:  "Origin",
			Value: "https://www.airnow.gov",
		},
		{
			Name:  "Connection",
			Value: "keep-alive",
		},
		{
			Name:  "Referer",
			Value: "https://www.airnow.gov/",
		},
		{
			Name:  "Pragma",
			Value: "no-cache",
		},
		{
			Name:  "Cache-Control",
			Value: "no-cache",
		},
	}
	queryParams = []KVPair{
		{
			Name:  "latitude",
			Value: "37.38029000000006",
		}, {
			Name:  "longitude",
			Value: "-122.08058499999999",
		}, {
			Name:  "stateCode",
			Value: "CA",
		}, {
			Name:  "maxDistance",
			Value: "50",
		},
	}
	responseHeaders = []KVPair{
		{
			Name:  "Cache-Control",
			Value: "public, max-age=3600",
		},
	}
)

func GetPrimaryDataSources(readings []APIReading) ([]APIReading, error) {
	var primarySources []APIReading
	for _, reading := range readings {
		if reading.IsPrimary && reading.ReportingArea == "Redwood City" && reading.DataType == "O" {
			primarySources = append(primarySources, reading)
		}
	}
	if len(primarySources) > 0 {
		return primarySources, nil
	} else {
		return primarySources, &NoPrimaryDataSourceError{}
	}
}

func GetAqiColorDesc(aqi int) (string, string) {
	if aqi == 0 {
		return "#7e0023", "ERROR"
	} else if aqi <= 50 {
		return "#00e400", "Good"
	} else if 50 <= aqi && aqi <= 100 {
		return "#ffff00", "Moderate"
	} else if 100 <= aqi && aqi <= 150 {
		return "#ff7e00", "Unsafe if Sensitive"
	} else if 150 <= aqi && aqi <= 200 {
		return "#ff0000", "Unhealthy"
	} else if 200 <= aqi && aqi <= 250 {
		return "#8F3F97", "Very Unhealthy"
	} else {
		return "#7e0023", "Hazardous"
	}
}

func getAQIFromServer() (int, error) {
	req, err := http.NewRequest("GET", airnowURL, nil)
	if err != nil {
		return 0, err
	}
	for _, header := range requestHeaders {
		req.Header.Add(header.Name, header.Value)
	}
	q := req.URL.Query()
	for _, param := range queryParams {
		q.Add(param.Name, param.Value)
	}
	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil
	}
	var (
		apiReadings      []APIReading
		latestDataSource APIReading
	)
	err = json.Unmarshal(respBody, &apiReadings)
	if err != nil {
		return 0, nil
	}

	// get latest issued reading
	primaryDataSources, err := GetPrimaryDataSources(apiReadings)
	for _, source := range primaryDataSources {
		if latestDataSource == (APIReading{}) {
			latestDataSource = source
			continue
		}
		curTime, err := time.Parse(timeFormat, source.IssueDate)
		if err != nil {
			return 0, err
		}
		newestTime, err := time.Parse(timeFormat, latestDataSource.IssueDate)
		if err != nil {
			return 0, err
		}
		if curTime.After(newestTime) {
			latestDataSource = source
		}
	}

	return latestDataSource.Aqi, nil
}

func handleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	aqi, err := getAQIFromServer()
	if err != nil {
		log.Fatalf("failed to get aqi %q", err)
	}
	aqiDescription, _ := GetAqiColorDesc(aqi)
	for _, header := range responseHeaders {
		w.Header().Set(header.Name, header.Value)
	}
	resp := AQIResp{AQI: aqi, Desc: aqiDescription}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		errorString := []byte(err.Error())
		w.WriteHeader(500)
		_, wErr := w.Write(errorString)
		if wErr != nil {
			log.Fatalf("failed to write error in response %q", wErr)
		}
	}
}

func handleImage(w http.ResponseWriter, r *http.Request) {
	imagick.Initialize()
	aqi, err := getAQIFromServer()

	aqiStr := strconv.Itoa(aqi)
	//imgLeftPadding := 150
	if err != nil {
		log.Fatalf("failed to get aqi %q", err)
	}
	color, desc := GetAqiColorDesc(aqi)
	for _, header := range responseHeaders {
		w.Header().Set(header.Name, header.Value)
	}
	mw := imagick.NewMagickWand()
	pw := imagick.NewPixelWand()
	dw := imagick.NewDrawingWand()
	// set background color from description map
	pw.SetColor(color)
	mw.NewImage(500, 500, pw)
	mw.SetFormat("png")
	dw.SetFont("Helvetica")
	dw.SetTextAntialias(true)
	dw.SetFontSize(80)
	pw.SetColor("#000000")
	dw.SetStrokeColor(pw)
	dw.SetGravity(imagick.GRAVITY_CENTER)
	// draw AQI header
	dw.Annotation(0, -175, "AQI")

	// draw AQI value
	dw.SetFontSize(160)
	dw.Annotation(0, 20, aqiStr)

	// draw description
	dw.SetFontSize(60)
	dw.Annotation(0, 200, desc)
	mw.DrawImage(dw)
	bytes := mw.GetImageBlob()
	w.Header().Set("mimetype", "image/PNG")
	w.Write(bytes)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/aqi", handleData).Methods("GET")
	r.HandleFunc("/image.png", handleImage).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("using default port")
	}
	log.Fatal(http.ListenAndServe(":"+port, r))
}
