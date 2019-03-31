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
)

type ignoreURLsRegExList []*regexp.Regexp
type removeParamsFromURLsRegExList []*regexp.Regexp

func (l ignoreURLsRegExList) IgnoreResource(url *url.URL) (bool, string) {
	URLtext := url.String()
	for _, regEx := range l {
		if regEx.MatchString(URLtext) {
			return true, fmt.Sprintf("Matched Ignore Rule `%s`", regEx.String())
		}
	}
	return false, ""
}

func (l removeParamsFromURLsRegExList) CleanResourceParams(url *url.URL) bool {
	// we try to clean all URLs, not specific ones
	return true
}

func (l removeParamsFromURLsRegExList) RemoveQueryParamFromResourceURL(paramName string) (bool, string) {
	for _, regEx := range l {
		if regEx.MatchString(paramName) {
			return true, fmt.Sprintf("Matched cleaner rule `%s`", regEx.String())
		}
	}

	return false, ""
}

type config struct {
	Generate                 bool          `docopt:"generate"`
	Hugo                     bool          `docopt:"hugo"`
	DestPath                 string        `docopt:"<destPath>"`
	From                     bool          `docopt:"from"`
	Dropmark                 bool          `docopt:"dropmark"`
	DropmarkURLs             []string      `docopt:"<url>"`
	CreateDestPath           bool          `docopt:"--create-dest-path"`
	HTTPUserAgent            string        `docopt:"--http-user-agent"`
	HTTPTimeout              time.Duration `docopt:"--http-timeout-secs"`
	SimulateScores           bool          `docopt:"--simulate-scores"`
	IgnoreURLsText           []string      `docopt:"--ignore-url"`
	RemoveParamsFromURLsText []string      `docopt:"--remove-param-from-url"`
	ShowConfig               bool          `docopt:"--show-config"`
	SaveErrorsInFile         string        `docopt:"--save-errors-in-file"`
	Verbose                  bool          `docopt:"-v,--verbose"`
	Summarize                bool          `docopt:"-s,--summarize"`

	errorsFile          *os.File
	errorsEncountered   bool
	ignoreURLs          ignoreURLsRegExList
	removeParamsFromURL removeParamsFromURLsRegExList
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
	if len(c.IgnoreURLsText) == 0 {
		c.IgnoreURLsText = []string{`^https://twitter.com/(.*?)/status/(.*)$`, `https://t.co`}
	}
	if len(c.RemoveParamsFromURLsText) == 0 {
		c.RemoveParamsFromURLsText = []string{`^utm_`}
	}
	c.ignoreURLs = make([]*regexp.Regexp, len(c.IgnoreURLsText))
	for i := 0; i < len(c.IgnoreURLsText); i++ {
		c.ignoreURLs[i] = regexp.MustCompile(c.IgnoreURLsText[i])
	}
	c.removeParamsFromURL = make([]*regexp.Regexp, len(c.RemoveParamsFromURLsText))
	for i := 0; i < len(c.RemoveParamsFromURLsText); i++ {
		c.removeParamsFromURL[i] = regexp.MustCompile(c.RemoveParamsFromURLsText[i])
	}
}

func (c *config) prepareDefaults() {
	c.prepareHTTPUserAgentDefault()
	c.prepareHTTPTimeoutDefault()
	c.prepareHarvesterDefaults()

	if len(c.SaveErrorsInFile) > 0 {
		f, err := os.Create(c.SaveErrorsInFile)
		if err != nil {
			panic(err)
		}
		c.errorsFile = f
	}
}

func (c *config) finish() {
	if c.errorsFile != nil {
		c.errorsFile.Close()
		if c.errorsEncountered && c.Verbose {
			fmt.Printf("Errors encountered and saved in %q.\n", c.SaveErrorsInFile)
		}
	}
}

func (c *config) showConfig() {
	fmt.Printf("DestPath: %q\n", c.DestPath)
	fmt.Printf("CreateDestPath: %v\n", c.CreateDestPath)
	fmt.Printf("SimulateScores: %v\n", c.SimulateScores)
	fmt.Printf("HTTPUserAgent: %q\n", c.HTTPUserAgent)
	fmt.Printf("HTTPTimeout: %d\n", c.HTTPTimeout)
	fmt.Printf("HarvesterIgnoreURLs: %+v\n", c.ignoreURLs)
	fmt.Printf("HarvesterRemoveParamsFromURLs: %+v\n", c.removeParamsFromURL)
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

func (c *config) reportErrors(errors []error) {
	if errors == nil {
		return
	}
	c.errorsEncountered = true
	if c.errorsFile != nil {
		for _, e := range errors {
			c.errorsFile.WriteString(fmt.Sprintf("* %v\n", e))
		}
	} else {
		for _, e := range errors {
			fmt.Printf("* %v\n", e)
		}
	}
}

var usage = `Lectio Control Utility.

Usage:
  lectioctl generate hugo <destPath> from dropmark <url>... [--save-errors-in-file=<file> --ignore-url=<iupattern>... --remove-param-from-url=<rparam>... --http-user-agent=<agent> --http-timeout-secs=<timeout> --create-dest-path --simulate-scores --show-config --verbose --summarize]

Options:
  -h --help                         Show this screen.
  --create-dest-path                Create the destination path if it doesn't already exist
  --http-user-agent=<agent>         The string to use for HTTP User-Agent header value
  --http-timeout-secs=<timeout>     How many seconds to wait before giving up on the HTTP request
  --simulate-scores                 Don't call Facebook, LinkedIn, etc. APIs; simulate the values instead
  --ignore-url=<iupattern>          A golang Regexp which instructs the harvester to ignore this URL pattern
  --remove-param-from-url=<rparam>  A golang Regexp which instructs the harvester to remove this param from URL query string
  --save-errors-in-file=<file>      If errors are found, save them to this file
  --show-config                     Show all config variables before running the utility
  -v --verbose                      Show verbose messages
  -s --summarize                    Summarize activity after execution
  --version                         Show version.`

func main() {
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

		for i := 0; i < len(options.DropmarkURLs); i++ {
			dropmarkURL := options.DropmarkURLs[i]
			collection, getErr := dropmark.GetDropmarkCollection(dropmarkURL, options.removeParamsFromURL, options.ignoreURLs, true, options.Verbose, options.HTTPUserAgent, options.HTTPTimeout)
			if getErr != nil {
				panic(getErr)
			}
			options.reportErrors(collection.Errors())
			generator := generator.NewHugoGenerator(collection, options.DestPath, options.Verbose, true)
			generator.GenerateContent()
			options.reportErrors(generator.Errors())
			if options.Summarize {
				fmt.Println(generator.GetActivitySummary())
			}
		}
	}

	options.finish()
}
