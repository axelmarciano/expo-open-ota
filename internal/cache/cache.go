package cache

import (
	"expo-open-ota/config"
	"sync"
)

type Cache interface {
	Get(key string) string
	Set(key string, value string, ttl *int) error
	Delete(key string)
	Clear() error
	TryLock(key string, ttl int) (bool, error)
	Sadd(key string, members []string, ttl *int) error
	Scard(key string) (int64, error)
}

type CacheType string

const (
	LocalCacheType CacheType = "local"
	RedisCacheType CacheType = "redis"
)

const defaultPrefix = "expoopenota"

func withPrefix(key string) string {
	prefix := config.GetEnv("CACHE_KEY_PREFIX")
	if prefix == "" {
		prefix = defaultPrefix
	}
	return prefix + ":" + key
}

func ResolveCacheType() CacheType {
	cacheType := config.GetEnv("CACHE_MODE")
	if cacheType == "redis" {
		return RedisCacheType
	}
	return LocalCacheType
}

var (
	cacheInstance Cache
	once          sync.Once
)

func GetCache() Cache {
	once.Do(func() {
		cacheType := ResolveCacheType()
		switch cacheType {
		case LocalCacheType:
			cacheInstance = NewLocalCache()
		case RedisCacheType:
			host := config.GetEnv("REDIS_HOST")
			password := config.GetEnv("REDIS_PASSWORD")
			port := config.GetEnv("REDIS_PORT")
			useTLS := config.GetEnv("REDIS_USE_TLS") == "true"
			// ACL and TLS configuration
			username := config.GetEnv("REDIS_USERNAME")
			caCertB64 := config.GetEnv("REDIS_CA_CERT_B64")
			cacheInstance = NewRedisCache(host, password, port, useTLS, username, caCertB64)
		default:
			panic("Unknown cache type")
		}
	})
	return cacheInstance
}
