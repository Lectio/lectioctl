package main

/****
 * TODO: Implement [Workbox](https://www.thepolyglotdeveloper.com/2019/03/service-workers-workbox-hugo-static-generated-site/)
 *       [Another implementation strategy](https://www.valleyease.me/2018/12/26/create-personal-pwa-site-with-hugo-and-webpack/)
 * TODO: Implement [JSON API (Custom Output Formats)](https://forestry.io/blog/build-a-json-api-with-hugo/)
 ****/

func main() {
	options := getUserOptions()

	if options.ShowConfig {
		options.showConfig()
	}

	if options.Generate && options.Hugo && options.From && options.Dropmark {
		generateHugoFromDropmark(options)
	}

	options.finish()
}
