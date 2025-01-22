package helpers

import "net/url"

func IsValidURL(str string) bool {
	parsedURL, err := url.Parse(str)
	if err != nil {
		return false
	}
	// Ensure the URL has a scheme (http, https, etc.) and a host
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}
