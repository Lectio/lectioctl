package main

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/lectio/dropmark"
	"github.com/lectio/generator"
	"github.com/lectio/harvester"
	"github.com/lectio/observe"
)

type ignoreURLsRegExList []*regexp.Regexp
type removeParamsFromURLsRegExList []*regexp.Regexp

func (l ignoreURLsRegExList) IgnoreDiscoveredResource(url *url.URL) (bool, string) {
	URLtext := url.String()
	for _, regEx := range l {
		if regEx.MatchString(URLtext) {
			return true, fmt.Sprintf("Matched Ignore Rule `%s`", regEx.String())
		}
	}
	return false, ""
}

func (l removeParamsFromURLsRegExList) CleanDiscoveredResource(url *url.URL) bool {
	// we try to clean all URLs, not specific ones
	return true
}

func (l removeParamsFromURLsRegExList) RemoveQueryParamFromResource(paramName string) (bool, string) {
	for _, regEx := range l {
		if regEx.MatchString(paramName) {
			return true, fmt.Sprintf("Matched cleaner rule `%s`", regEx.String())
		}
	}

	return false, ""
}

type config struct {
	Generate                          bool          `docopt:"generate"`
	Hugo                              bool          `docopt:"hugo"`
	DestPath                          string        `docopt:"<destPath>"`
	From                              bool          `docopt:"from"`
	Dropmark                          bool          `docopt:"dropmark"`
	DropmarkURLs                      []string      `docopt:"<url>"`
	CreateDestPath                    bool          `docopt:"--create-dest-path"`
	HTTPUserAgent                     string        `docopt:"--http-user-agent"`
	HTTPTimeout                       time.Duration `docopt:"--http-timeout-secs"`
	SimulateScores                    bool          `docopt:"--simulate-scores"`
	HarvesterIgnoreURLsText           []string      `docopt:"--harvester-ignore-url"`
	HarvesterRemoveParamsFromURLsText []string      `docopt:"--harvester-remove-param-from-url"`
	ShowConfig                        bool          `docopt:"--show-config"`
	Verbose                           bool          `docopt:"-v,--verbose"`
	Summarize                         bool          `docopt:"-s,--summarize"`

	harvesterIgnoreURLs          ignoreURLsRegExList
	harvesterRemoveParamsFromURL removeParamsFromURLsRegExList
}

func (c *config) prepareHTTPUserAgentDefault() {
	if len(c.HTTPUserAgent) == 0 {
		c.HTTPUserAgent = "github.com/lectio/lectioctl"
	}
}

func (c *config) prepareHTTPTimeoutDefault() {
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = time.Second * 90
	} else {
		c.HTTPTimeout = time.Second * c.HTTPTimeout
	}
}

func (c *config) prepareHarvesterDefaults() {
	if len(c.HarvesterIgnoreURLsText) == 0 {
		c.HarvesterIgnoreURLsText = []string{`^https://twitter.com/(.*?)/status/(.*)$`, `https://t.co`}
	}
	if len(c.HarvesterRemoveParamsFromURLsText) == 0 {
		c.HarvesterRemoveParamsFromURLsText = []string{`^utm_`}
	}
	c.harvesterIgnoreURLs = make([]*regexp.Regexp, len(c.HarvesterIgnoreURLsText))
	for i := 0; i < len(c.HarvesterIgnoreURLsText); i++ {
		c.harvesterIgnoreURLs[i] = regexp.MustCompile(c.HarvesterIgnoreURLsText[i])
	}
	c.harvesterRemoveParamsFromURL = make([]*regexp.Regexp, len(c.HarvesterRemoveParamsFromURLsText))
	for i := 0; i < len(c.HarvesterRemoveParamsFromURLsText); i++ {
		c.harvesterRemoveParamsFromURL[i] = regexp.MustCompile(c.HarvesterRemoveParamsFromURLsText[i])
	}
}

func (c *config) prepareDefaults() {
	c.prepareHTTPUserAgentDefault()
	c.prepareHTTPTimeoutDefault()
	c.prepareHarvesterDefaults()
}

func (c *config) showConfig() {
	fmt.Printf("DestPath: %q\n", c.DestPath)
	fmt.Printf("CreateDestPath: %v\n", c.CreateDestPath)
	fmt.Printf("SimulateScores: %v\n", c.SimulateScores)
	fmt.Printf("HTTPUserAgent: %q\n", c.HTTPUserAgent)
	fmt.Printf("HTTPTimeout: %d\n", c.HTTPTimeout)
	fmt.Printf("HarvesterIgnoreURLs: %+v\n", c.harvesterIgnoreURLs)
	fmt.Printf("HarvesterRemoveParamsFromURLs: %+v\n", c.harvesterRemoveParamsFromURL)
}

func (c config) createDestPathIfNotExists() {
	if !c.CreateDestPath {
		return
	}
	created, err := CreateDirIfNotExist(c.DestPath)
	if err != nil {
		panic(err)
	}
	if created && c.Verbose {
		fmt.Printf("Created directory %q\n", c.DestPath)
	}
}

var usage = `Lectio Control Utility.

Usage:
  lectioctl generate hugo <destPath> from dropmark <url>... [--harvester-ignore-url=<hiURL>... --harvester-remove-param-from-url=<hrparam>... --http-user-agent=<agent> --http-timeout-secs=<timeout> --create-dest-path --simulate-scores --show-config --verbose --summarize]

Options:
  -h --help                                   Show this screen.
  --create-dest-path                          Create the destination path if it doesn't already exist
  --http-user-agent=<agent>                   The string to use for HTTP User-Agent header value
  --http-timeout-secs=<timeout>               How many seconds to wait before giving up on the HTTP request
  --simulate-scores                           Don't call Facebook, LinkedIn, etc. APIs; simulate the values instead
  --harvester-ignore-url=<hiURL>              A golang Regexp which instructs the harvester to ignore this URL pattern
  --harvester-remove-param-from-url=<hrparam> A golang Regexp which instructs the harvester to remove this param from URL query string
  --show-config                               Show all config variables before running the utility
  -v --verbose                                Show verbose messages
  -s --summarize                              Summarize activity after execution
  --version                                   Show version.`

func main() {
	_, set := os.LookupEnv("JAEGER_SERVICE_NAME")
	if !set {
		os.Setenv("JAEGER_SERVICE_NAME", "Lectio Control Utility")
	}

	observatory := observe.MakeObservatoryFromEnv()
	span := observatory.StartTrace("lectioctl")

	arguments, pdErr := docopt.ParseDoc(usage)
	if pdErr != nil {
		panic(pdErr)
	}
	options := new(config)
	bindErr := arguments.Bind(options)
	if bindErr != nil {
		fmt.Printf("%+v, %v", options, bindErr)
		panic(pdErr)
	}
	options.prepareDefaults()

	if options.ShowConfig {
		options.showConfig()
	}

	if options.Generate && options.Hugo && options.From && options.Dropmark {
		options.createDestPathIfNotExists()

		ch := harvester.MakeContentHarvester(observatory, options.harvesterIgnoreURLs, options.harvesterRemoveParamsFromURL, true)

		for i := 0; i < len(options.DropmarkURLs); i++ {
			dropmarkURL := options.DropmarkURLs[i]
			collection, getErr := dropmark.GetDropmarkCollection(ch, span, options.Verbose, dropmarkURL, options.HTTPUserAgent, options.HTTPTimeout)
			if getErr != nil {
				panic(getErr)
			}
			if collection.Errors() != nil {
				fmt.Println(collection.Errors())
			}
			generator := generator.NewHugoGenerator(collection, options.DestPath, options.Verbose, true)
			generator.GenerateContent()
			if generator.Errors() != nil {
				fmt.Println(generator.Errors())
			}
			if options.Summarize {
				fmt.Println(generator.GetActivitySummary())
			}
		}
	}
}
