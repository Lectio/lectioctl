package main

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/lectio/content"
	"github.com/lectio/score"

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
	HugoHomePath             string        `docopt:"<hugoHomePath>"`
	HugoContentID            string        `docopt:"<hugoContentID>"`
	From                     bool          `docopt:"from"`
	Dropmark                 bool          `docopt:"dropmark"`
	DropmarkURLs             []string      `docopt:"<url>"`
	CreateDestPaths          bool          `docopt:"--create-dest-paths"`
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
	fmt.Printf("HugoHomePath: %q\n", c.HugoHomePath)
	fmt.Printf("HugoContentID: %q\n", c.HugoContentID)
	fmt.Printf("CreateDestPaths: %v\n", c.CreateDestPaths)
	fmt.Printf("SimulateScores: %v\n", c.SimulateScores)
	fmt.Printf("HTTPUserAgent: %q\n", c.HTTPUserAgent)
	fmt.Printf("HTTPTimeout: %d\n", c.HTTPTimeout)
	fmt.Printf("HarvesterIgnoreURLs: %+v\n", c.ignoreURLs)
	fmt.Printf("HarvesterRemoveParamsFromURLs: %+v\n", c.removeParamsFromURL)
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
  lectioctl generate hugo <hugoHomePath> <hugoContentID> from dropmark <url>... [--save-errors-in-file=<file> --ignore-url=<iupattern>... --remove-param-from-url=<rparam>... --http-user-agent=<agent> --http-timeout-secs=<timeout> --create-dest-paths --simulate-scores --show-config --verbose]

Options:
  -h --help                         Show this screen.
  --create-dest-paths               Create the destination path(s) if they doesn't already exist
  --http-user-agent=<agent>         The string to use for HTTP User-Agent header value
  --http-timeout-secs=<timeout>     How many seconds to wait before giving up on the HTTP request
  --simulate-scores                 Don't call Facebook, LinkedIn, etc. APIs; simulate the values instead
  --ignore-url=<iupattern>          A golang Regexp which instructs the harvester to ignore this URL pattern
  --remove-param-from-url=<rparam>  A golang Regexp which instructs the harvester to remove this param from URL query string
  --save-errors-in-file=<file>      If errors are found, save them to this file
  --show-config                     Show all config variables before running the utility
  -v --verbose                      Show verbose messages
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
		for i := 0; i < len(options.DropmarkURLs); i++ {
			dropmarkURL := options.DropmarkURLs[i]
			dropmarkCollection, getErr := dropmark.GetDropmarkCollection(dropmarkURL, options.removeParamsFromURL, options.ignoreURLs, true, options.Verbose, options.HTTPUserAgent, options.HTTPTimeout)
			if getErr != nil {
				panic(getErr)
			}
			options.reportErrors(dropmarkCollection.Errors())

			filterResults := dropmarkCollection.FilterInvalidItems()
			options.reportErrors(filterResults.Errors())

			fcItems, fcItemsErr := filterResults.Filtered().Content()
			if fcItemsErr != nil {
				panic(fcItemsErr)
			}

			handler := func(index int) (*url.URL, string, error) {
				item := fcItems[index].(content.CuratedContent)
				url, urlErr := item.Link().FinalURL()
				if urlErr != nil {
					return url, item.Keys().GloballyUniqueKey(), urlErr
				}
				return url, item.Keys().GloballyUniqueKey(), nil
			}
			iterator := func() (startIndex int, endIndex int, retrievalFn score.TargetsIteratorRetrievalFn) {
				return 0, len(fcItems) - 1, handler
			}
			sc := score.MakeCollection(iterator, options.Verbose, true)

			options.reportErrors(sc.Errors())
			generator, genErr := generator.NewHugoGenerator(filterResults.Filtered(), sc, options.HugoHomePath, options.HugoContentID, options.CreateDestPaths, options.Verbose, true)
			if genErr != nil {
				panic(genErr)
			}
			generator.GenerateContent()
			options.reportErrors(generator.Errors())
		}
	}

	options.finish()
}
