package main

import (
	"fmt"
	"net/url"
	"regexp"
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
