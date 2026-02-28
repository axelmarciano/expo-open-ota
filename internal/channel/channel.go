package channel

import (
	"expo-open-ota/internal/bucket"
)

func FetchChannels() ([]string, error) {
	resolvedBucket := bucket.GetBucket()
	return resolvedBucket.GetChannels()
}

func GetChannelMapping(channelName string) (string, error) {
	resolvedBucket := bucket.GetBucket()
	return resolvedBucket.GetChannelMapping(channelName)
}

func SetChannelMapping(channelName, branch string) error {
	resolvedBucket := bucket.GetBucket()
	return resolvedBucket.SetChannelMapping(channelName, branch)
}

func DeleteChannel(channelName string) error {
	resolvedBucket := bucket.GetBucket()
	return resolvedBucket.DeleteChannel(channelName)
}
