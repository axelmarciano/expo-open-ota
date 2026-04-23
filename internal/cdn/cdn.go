package cdn

import "sync"

type CDN interface {
	isCDNAvailable() bool
	// ComputeRedirectionURLForAsset mirrors the v2 bucket layout:
	// {keyPrefix}{appId}/{branch}/{runtimeVersion}/{updateId}/{asset}.
	// Missing the appId segment produces a 404 against S3/CloudFront and
	// GCS-direct (objects are not at the top level anymore).
	ComputeRedirectionURLForAsset(appId, branch, runtimeVersion, updateId, asset string) (string, error)
}

var (
	cdnInstance CDN
	once        sync.Once
)

func GetCDN() CDN {
	once.Do(func() {
		cloudfrontCDN := CloudfrontCDN{}
		isCloudfrontCDNavailable := (&cloudfrontCDN).isCDNAvailable()
		if isCloudfrontCDNavailable {
			cdnInstance = &cloudfrontCDN
		} else {
			gcsCDN := GCSDirectCDN{}
			if (&gcsCDN).isCDNAvailable() {
				cdnInstance = &gcsCDN
			}
		}
	})
	return cdnInstance
}

func ResetCDNInstance() {
	cdnInstance = nil
	once = sync.Once{}
}
