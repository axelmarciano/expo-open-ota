package cache

import (
	"expo-open-ota/config"
	"strings"
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

const (
	RedisSentinelCacheType CacheType = "redis-sentinel"
)

func ResolveCacheType() CacheType {
	cacheType := config.GetEnv("CACHE_MODE")
	switch cacheType {
	case "redis":
		return RedisCacheType
	case "redis-sentinel":
		return RedisSentinelCacheType
	default:
		return LocalCacheType
	}
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
			username := config.GetEnv("REDIS_USERNAME")
			caCertB64 := config.GetEnv("REDIS_CA_CERT_B64")
			cacheInstance = NewRedisCache(host, password, port, useTLS, username, caCertB64)
		case RedisSentinelCacheType:
			sentinelAddrsStr := config.GetEnv("REDIS_SENTINEL_ADDRS")
			sentinelAddrs := strings.Split(sentinelAddrsStr, ",")
			masterName := config.GetEnv("REDIS_SENTINEL_MASTER_NAME")
			if masterName == "" {
				masterName = "mymaster"
			}
			password := config.GetEnv("REDIS_PASSWORD")
			useTLS := config.GetEnv("REDIS_USE_TLS") == "true"
			username := config.GetEnv("REDIS_USERNAME")
			caCertB64 := config.GetEnv("REDIS_CA_CERT_B64")
			cacheInstance = NewRedisSentinelCache(sentinelAddrs, masterName, password, useTLS, username, caCertB64)
		default:
			panic("Unknown cache type")
		}
	})
	return cacheInstance
}
