package main

import (
	"net/url"

	"github.com/lectio/content"
	"github.com/lectio/dropmark"
	"github.com/lectio/generator"
	"github.com/lectio/score"
)

func generateHugoFromDropmark(options *config) {
	for i := 0; i < len(options.DropmarkURLs); i++ {
		dropmarkURL := options.DropmarkURLs[i]
		dropmarkCollection, getErr := dropmark.GetDropmarkCollection(dropmarkURL, options.nlp, options.removeParamsFromURL, options.ignoreURLs, true, options.makeProgressReporter(), options.HTTPUserAgent, options.HTTPTimeout)
		if getErr != nil {
			panic(getErr)
		}
		options.reportErrors(dropmarkCollection.Errors())

		// TODO: add duplicates detection as part of "invalid items" filter
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
		sc := score.MakeCollection(iterator, options.makeProgressReporter(), true)

		// TODO: implement [recommendation system](https://medium.com/@williamscott701/pinsage-how-pinterest-improved-their-recommendation-system-149cb35fdfa5)

		options.reportErrors(sc.Errors())
		generator, genErr := generator.NewHugoGenerator(filterResults.Filtered(), sc, options.HugoHomePath, options.HugoContentID, options.CreateDestPaths, options.makeProgressReporter(), true)
		if genErr != nil {
			panic(genErr)
		}
		generator.GenerateContent()
		options.reportErrors(generator.Errors())
	}
}
