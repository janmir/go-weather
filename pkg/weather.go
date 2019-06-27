package weather

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/janmir/go-util"
)

const (
	_weatherURLGoogle        = "https://www.google.com/search?q=%s%%20weather&hl=en"
	_weatherURLApixuForecast = "https://api.apixu.com/v1/forecast.json?key=%s&q=%s&days=10&"
	_weatherURLApixuHistory  = "https://api.apixu.com/v1/history.json?key=%s&q=%s&dt=%s"

	//website formatting changes based on the request
	//user-agent value, need to implicitly set use-agent
	//to get expected formatting
	_mobileUserAgent  = "Mozilla/5.0 (Linux; U; Android 4.4.2; en-us; SCH-I535 Build/KOT49H) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
	_desktopUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36"

	//Selectors
	_container = "div#wob_wc"                          // container
	_summary   = "div#wob_wc > span[role=heading] > *" // location, date, summary
	_detailed  = "div#wob_wc > div#wob_d > div > *"    // image, temps, precipitation/wind/humid
	_sub       = "div#wob_wc .wob_df > *"              // image, temps, precipitation/wind/humid
)

var (
	hilo = regexp.MustCompile(`(?P<hi>\d{2})\d{2}°(?P<lo>\d{2})\d{2}°`)
)

//Weather holds the weather data
//and methods
type Weather struct {
	//private
	client http.Client

	//Public Data
	Location      string
	Date          string
	Summary       string
	Image         string
	Temp          string
	Wind          string
	Precipitation string
	Humidity      string
	Sub           []SubWeather
	NotAvailable  bool // Whether the data souce is historical and currently not available
	Emoji         string
}

// SubWeather contains a summary/brief weather
// forecast for current/historic data with latest
// forecast included.
type SubWeather struct {
	Date    string
	Summary string
	Emoji   string
	TempHi  string
	TempLo  string
}

//New initializes the variables
//used by weather struct
func New() Weather {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	return Weather{
		client: client,
	}
}

// String is the stringer
func (w *Weather) String() string {
	return fmt.Sprintf("Weather in %s is: %s.\n",
		w.Location, w.Summary)
}

// Get retrieves the forecast on varrying sources
// based on the date given
func (w *Weather) Get(city string, date time.Time) {
	today := time.Now()
	diff := date.Sub(today).Round(time.Hour).Hours() / 24
	util.Logger("Diff:", diff)

	// Check time is within the next 7 days or more
	switch {

	// Forecast and Historical data should be cached.
	case diff > 10: // > 10 apixu historical
		util.Red("FROM APIXU-HISTORY")

		// Free accounts can only access 7 days prior
		// data.
		//w.GetPixuHistorical(city, date)

		// Set not available flag
		w.NotAvailable = true
	case diff > 8: // > 7 apixu forecast
		util.Green("FROM APIXU-FORECAST")
		w.GetPixuForecast(city)
	default: // If within the next 7 days used google weather results
		util.Green("FROM GOOGLE")
		w.GetGoogle(city)
	}
}

// GetPixuForecast ...
func (w *Weather) GetPixuForecast(city string) {
	//url encode city
	city = url.QueryEscape(city)

	// Construct the path
	path := fmt.Sprintf(_weatherURLApixuForecast,
		_apixuKey, city)

	// Query
	w.GetPixu(path)
}

// GetPixuHistorical ...
func (w *Weather) GetPixuHistorical(city string, date time.Time) {
	//url encode city
	city = url.QueryEscape(city)

	// Subtract a year from date
	oldDate := date.AddDate(-1, 0, 0)
	strDate := url.QueryEscape(oldDate.Format("2006-01-02"))

	// Construct the path
	path := fmt.Sprintf(_weatherURLApixuHistory,
		_apixuKey, city, strDate)

	// Query
	w.GetPixu(path)
}

// GetPixu ...
func (w *Weather) GetPixu(cleanPath string) {
	req, err := http.NewRequest("GET", cleanPath, nil)
	util.Catch(err)

	//set user-agent
	req.Header.Set("user-agent", _desktopUserAgent)

	res, err := w.client.Do(req)
	util.HTTPCatch(res, err, "Unable to fetch data from url:", cleanPath)

	/***Debugging: Dump response***
	b, err := ioutil.ReadAll(res.Body)
	util.Catch(err)
	defer res.Body.Close()
	fmt.Printf("%s", string(b))
	/****/

	//Parse the html data
	err = w.parsePixu(res.Body)
	util.Catch(err)
}

func (w *Weather) parsePixu(body io.ReadCloser) error {
	defer body.Close()

	// Unmarshal the data
	b, err := ioutil.ReadAll(body)
	util.Catch(err)

	pixu := &Apixu{}
	err = json.Unmarshal(b, pixu)
	util.Catch(err, "Unable to parse Apixu json data: ", string(b))

	/*
		Location      string
		Date          string
		Summary       string
		Image         string
		Temp          string
		Wind          string
		Precipitation string
		Humidity      string
		Emoji 		  string
	*/

	w.Location = pixu.Location.Name
	w.Date = pixu.Location.Localtime
	w.Summary = pixu.Current.Condition.Text
	w.Image = pixu.Current.Condition.Icon
	w.Temp = fmt.Sprintf("%0.2f°", pixu.Current.TempC)
	w.Wind = fmt.Sprintf("%0.2fKph", pixu.Current.WindKph)
	w.Precipitation = fmt.Sprintf("%0.2fmm", pixu.Current.PrecipMm)
	w.Humidity = fmt.Sprintf("%d", pixu.Current.Humidity)
	w.Emoji = w.getEmoji(w.Summary)

	return nil
}

func (w *Weather) getEmoji(condition string) string {
	emoji := "<<emoji_here>>"

	// Convert to lowercase
	condition = strings.ToLower(condition)
	switch {
	case strings.Contains(condition, "clear"):
		fallthrough
	case strings.Contains(condition, "sun"): //sunny, cloudy-sunny
		if strings.Contains(condition, "cloud") {
			emoji = emojiMap["cloudy-sunny"]
		} else {
			emoji = emojiMap["sunny"]
		}
	case strings.Contains(condition, "rain"): //rain, cloudy-rain
		if strings.Contains(condition, "cloud") {
			emoji = emojiMap["cloudy-rain"]
		} else {
			emoji = emojiMap["rain"]
		}
	case strings.Contains(condition, "cloud"): //thunder
		emoji = emojiMap["cloudy"]
	case strings.Contains(condition, "thunder"): //thunder
		emoji = emojiMap["thunder"]
	case strings.Contains(condition, "snow"): //snow
		emoji = emojiMap["snow"]
	}
	return emoji
}

//GetGoogle retrieves the latest weather data
//from google
func (w *Weather) GetGoogle(city string) {
	//url encode city
	city = url.QueryEscape(city)

	path := fmt.Sprintf(_weatherURLGoogle, city)
	req, err := http.NewRequest("GET", path, nil)
	util.Catch(err)

	//set user-agent
	req.Header.Set("user-agent", _desktopUserAgent)

	res, err := w.client.Do(req)
	util.HTTPCatch(res, err, "Unable to fetch data from url:", path)

	/***Debugging***
	//read the response body
	b, err := ioutil.ReadAll(res.Body)
	util.Catch(err)
	defer res.Body.Close()
	util.Logger("%s", string(b))
	/****/

	//Parse the html data
	err = w.parseGoogle(res.Body)
	util.Catch(err)

}

//parseGoogle function may change a lot of times due to
//changes in internal structure or layout of
//the weather data shown by google
func (w *Weather) parseGoogle(body io.ReadCloser) error {
	defer body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(body)
	util.Catch(err, "Unable to parse the HTML string.")

	// Check first if the parent container exist
	len := doc.Find(_container).Length()
	//util.Logger("length: ", len)

	//get data if container exists
	if len == 1 {
		//summary data [top-left area]
		doc.Find(_summary).Each(func(i int, node *goquery.Selection) {
			val := node.Text()

			switch i {
			case 0: //Location
				w.Location = val
			case 1: //Date
				w.Date = val
			case 2: //Summary
				w.Summary = val
				w.Emoji = w.getEmoji(val)
			}
		})

		//detailed data [top-right area]
		doc.Find(_detailed).Each(func(i int, node *goquery.Selection) {
			switch i {
			case 0: //Image
				src, ok := node.Attr("src")
				if ok {
					w.Image = "https:" + src
				}
			case 1: //Temp
				val := strings.TrimSpace(node.Find("#wob_tm").Text())
				w.Temp = val + "°C"
			case 2: //Precipitation/Humidity/Wind
				node.Find("div > span:first-child").Each(func(i int, node *goquery.Selection) {
					val := strings.TrimSpace(node.Text())
					switch i {
					case 0: //Precipitation
						w.Precipitation = val
					case 1: //Humidity
						w.Humidity = val
					case 2: //Wind
						val = strings.TrimSpace(node.Find("span:first-child").Text())
						w.Wind = val
					}
				})
			}
		})

		// Get Sub-Weather data
		subs := doc.Find(_sub)
		if subs.Length()%3 == 0 {
			var subWeather SubWeather
			subs.Each(func(i int, node *goquery.Selection) {
				val := strings.TrimSpace(node.Text())

				switch i % 3 {
				case 0: // Date
					subWeather = SubWeather{}
					subWeather.Date = val
				case 1: // Summary
					img := node.Find("img")
					val = strings.TrimSpace(img.AttrOr("alt", "<Unknown>"))
					subWeather.Summary = val
					subWeather.Emoji = w.getEmoji(val)
				case 2: // Temp HI/LO
					// parse the text to separate
					// hi and low values
					temps := util.MatchToMap(hilo, val)

					subWeather.TempHi = temps["hi"] + "°"
					subWeather.TempLo = temps["lo"] + "°"

					// Last entry append here
					w.Sub = append(w.Sub, subWeather)
				}
			})
		}

		return nil
	}

	return fmt.Errorf("Error in parsing the HTML string, container %s does not exist",
		_container)
}
