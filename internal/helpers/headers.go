package helpers

import (
	"net/http"
	"strings"
)

func ParseExpoExtraParams(header string) map[string]string {
	params := make(map[string]string)
	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(parts[1], `"`)
		params[key] = value
	}
	return params
}

func GetChannelOverride(headers http.Header) string {
	channelName := ""
	extra := headers.Get("expo-extra-params")
	if extra != "" {
		params := ParseExpoExtraParams(extra)
		if override, ok := params["ow-expo-channel"]; ok && override != "" {
			if override != headers.Get("expo-channel-name") {
				channelName = override
			}
		}
	}
	return channelName
}

func ResolveExpoChannel(headers http.Header) string {
	channelName := headers.Get("expo-channel-name")
	if channelName == "" {
		return ""
	}
	channelOverride := GetChannelOverride(headers)
	if channelOverride != "" {
		channelName = channelOverride
	}
	return channelName
}
