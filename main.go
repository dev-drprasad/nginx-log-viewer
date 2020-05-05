package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	useragent "github.com/mssola/user_agent"
)

// HTTPEmojis is statuscodes and emoji map
var HTTPEmojis = map[int]string{
	http.StatusOK:                  "🎉",
	http.StatusCreated:             "🔨",
	http.StatusMovedPermanently:    "👉",
	http.StatusBadRequest:          "👎",
	http.StatusFound:               "👉",
	http.StatusUnauthorized:        "✋",
	http.StatusNotFound:            "🤷",
	http.StatusInternalServerError: "😱",
	http.StatusGatewayTimeout:      "⌛",
}

// HTTPColors is statucode and color map
var HTTPColors = map[int]aurora.Color{
	http.StatusOK:                  aurora.WhiteFg | aurora.GreenBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusCreated:             aurora.WhiteFg | aurora.GreenBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusMovedPermanently:    aurora.WhiteFg | aurora.CyanBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusBadRequest:          aurora.WhiteFg | aurora.BrownBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusFound:               aurora.WhiteFg | aurora.BrownBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusUnauthorized:        aurora.WhiteFg | aurora.BrownBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusNotFound:            aurora.WhiteFg | aurora.BrownBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusInternalServerError: aurora.WhiteFg | aurora.RedBg | aurora.BoldFm | aurora.BrightBg,
	http.StatusGatewayTimeout:      aurora.WhiteFg | aurora.RedBg | aurora.BoldFm | aurora.BrightBg,
}

var varToRegexReplacer = strings.NewReplacer("$time_iso8601", `(?P<Time>.*)`,
	"$http_x_username", `(?P<HTTPXUsername>.*)`,
	"$status", `(?P<Status>\d+)`, "$upstream_addr", "(?P<UpstreamAddr>.*)", "$http_user_agent", "(?P<UserAgent>.*)",
	"$request", `(?P<Method>.*) (?P<URL>.*) HTTP/(?P<HTTPVersion>\d.\d)`, "[", `\[`, "]", `\]`)

var accessLogFormat = `[$time_iso8601] $http_x_username $status "$request" $upstream_addr "$http_user_agent"`
var accessLogRegex = varToRegexReplacer.Replace(accessLogFormat) + "$"
var errorLogRegex = ` \[.+\] .* \*\d+ (?P<Message>.*), client:.*, server:.*`

const timeFormat = "02 January 2006, 03:04:05 PM"

type logline struct {
	Time          time.Time
	HTTPXUsername string
	Status        int
	Method        string
	URL           string
	HTTPVersion   string
	UpstreamAddr  string
	UserAgent     *useragent.UserAgent
	Message       string
}

var au aurora.Aurora

func init() {
	au = aurora.NewAurora(true)
}

func main() {
	accessLogRe := regexp.MustCompile(accessLogRegex)
	errorLogRe := regexp.MustCompile(errorLogRegex)

	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if info.Mode()&os.ModeCharDevice != 0 {
		fmt.Println("The command is intended to work with pipes.")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _, err := reader.ReadLine()
		if err != nil && err == io.EOF {
			break
		}
		inputStr := string(input)

		l := logline{}
		if strings.Contains(inputStr, " [error] ") {
			match := errorLogRe.FindStringSubmatch(inputStr)

			if len(match) == 2 {
				l.Message = match[1]
			}
			input, _, err := reader.ReadLine()
			if err != nil && err == io.EOF {
				break
			}
			inputStr = string(input)
		}

		match := accessLogRe.FindStringSubmatch(inputStr)

		for i, name := range accessLogRe.SubexpNames() {
			if i > 0 && i <= len(match) {
				switch name {
				case "Status":
					v, _ := strconv.Atoi(match[i])
					l.Status = v
				case "Time":
					t, _ := time.Parse(time.RFC3339, match[i])
					l.Time = t
				case "UserAgent":
					l.UserAgent = useragent.New(match[i])
				default:
					reflect.ValueOf(&l).Elem().FieldByName(name).SetString(match[i])
				}
			}
		}

		fmt.Printf("%s\n", au.Gray(11, l.Time.Format(timeFormat)))
		fmt.Printf("🤡  %s\n", au.Yellow(l.HTTPXUsername))
		emoji, ok := HTTPEmojis[l.Status]
		if !ok {
			emoji = "🤷"
		}
		color, ok := HTTPColors[l.Status]
		if !ok {
			color = aurora.WhiteFg | aurora.YellowBg | aurora.BoldFm
		}
		fmt.Printf("%s  %s  %s  %s\n", emoji, au.Colorize(fmt.Sprintf(" %d ", l.Status), color), au.BrightCyan(l.Method).Bold(), au.White(l.URL))

		if l.UserAgent != nil {
			browser, browserVersion := l.UserAgent.Browser()
			fmt.Printf("🌎  %s %s  🖥️  %s\n", au.Magenta(browser).Bold(), au.Gray(11, "("+browserVersion+")"), au.Magenta(l.UserAgent.OSInfo().Name).Bold())
		}

		if l.Message != "" {
			fmt.Printf("📩  %s\n", au.Red(l.Message))
		}

		fmt.Println()
	}

}

// https://stackoverflow.com/questions/47961245
// https://stackoverflow.com/questions/30483652
// https://stackoverflow.com/questions/40512323
