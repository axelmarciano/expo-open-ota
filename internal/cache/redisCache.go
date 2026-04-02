package cache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client redis.UniversalClient
}

func buildTLSConfig(caCertB64 string) *tls.Config {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if caCertB64 != "" {
		caCertPEM, err := base64.StdEncoding.DecodeString(caCertB64)
		if err != nil {
			log.Printf("Failed to decode CA certificate from base64: %v", err)
		} else {
			certPool := x509.NewCertPool()
			if certPool.AppendCertsFromPEM(caCertPEM) {
				tlsConfig.RootCAs = certPool
			} else {
				log.Printf("Failed to append CA certificate to pool")
			}
		}
	}
	return tlsConfig
}

func NewRedisCache(host, password, port string, useTLS bool, username, caCertB64 string) *RedisCache {
	opts := &redis.Options{
		Addr:     host + ":" + port,
		Password: password,
	}
	if username != "" {
		opts.Username = username
	}
	if useTLS {
		opts.TLSConfig = buildTLSConfig(caCertB64)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		panic(err)
	}

	return &RedisCache{client: client}
}

func NewRedisSentinelCache(sentinelAddrs []string, masterName, password string, useTLS bool, username, caCertB64 string) *RedisCache {
	opts := &redis.FailoverOptions{
		SentinelAddrs: sentinelAddrs,
		MasterName:    masterName,
		Password:      password,
	}
	if username != "" {
		opts.Username = username
	}
	if useTLS {
		opts.TLSConfig = buildTLSConfig(caCertB64)
	}

	client := redis.NewFailoverClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		panic(fmt.Sprintf("Redis Sentinel connection failed: %v", err))
	}

	log.Printf("Connected to Redis via Sentinel (master: %s)", masterName)
	return &RedisCache{client: client}
}

func (c *RedisCache) Get(key string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := c.client.Get(ctx, withPrefix(key)).Result()
	if errors.Is(err, redis.Nil) {
		return ""
	} else if err != nil {
		return ""
	}
	return val
}

func (c *RedisCache) Set(key string, value string, ttl *int) error {
	expiration := time.Duration(0)
	if ttl != nil {
		expiration = time.Duration(*ttl) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return c.client.Set(ctx, withPrefix(key), value, expiration).Err()
}

func (c *RedisCache) Delete(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c.client.Del(ctx, withPrefix(key))
}

func (c *RedisCache) Clear() error {
	fmt.Println("Cache can only be cleared in development mode.")
	return nil
}

func (r *RedisCache) TryLock(key string, ttl int) (bool, error) {
	ctx := context.Background()
	ok, err := r.client.SetNX(ctx, withPrefix(key), "locked", time.Duration(ttl)*time.Second).Result()
	return ok, err
}

func (c *RedisCache) Sadd(key string, members []string, ttl *int) error {
	if len(members) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fullKey := withPrefix(key)

	vals := make([]interface{}, len(members))
	for i, m := range members {
		vals[i] = m
	}

	added, err := c.client.SAdd(ctx, fullKey, vals...).Result()
	if err != nil {
		return err
	}

	if ttl != nil && added > 0 {
		_ = c.client.Expire(ctx, fullKey, time.Duration(*ttl)*time.Second).Err()
	}

	return nil
}

func (c *RedisCache) Scard(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return c.client.SCard(ctx, withPrefix(key)).Result()
}
