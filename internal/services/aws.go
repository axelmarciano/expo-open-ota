package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"expo-open-ota/config"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/cloudfront/sign"
	"log"
	"sync"
	"time"
)

var (
	s3Client     *s3.Client
	initS3Client sync.Once
)

func GetS3Client() (*s3.Client, error) {
	var err error

	initS3Client.Do(func() {
		var cfg aws.Config
		cfg, err = awsconfig.LoadDefaultConfig(context.TODO())
		if err == nil {
			s3Client = s3.NewFromConfig(cfg)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("erreur lors du chargement de la config AWS: %w", err)
	}
	return s3Client, nil
}

func FetchSecret(secretName string) string {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}

	client := secretsmanager.NewFromConfig(cfg)

	resp, err := client.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		log.Fatalf("Failed to retrieve secret %s: %v", secretName, err)
	}

	if resp.SecretString == nil {
		log.Fatalf("Secret %s has no SecretString", secretName)
	}

	return *resp.SecretString
}

func GenerateSignedCookies(resource string, cloudFrontPrivateKey string) (string, error) {
	keyPairID := config.GetEnv("CLOUDFRONT_KEY_PAIR_ID")
	if keyPairID == "" {
		return "", errors.New("CLOUDFRONT_KEY_PAIR_ID not set in environment")
	}

	block, _ := pem.Decode([]byte(cloudFrontPrivateKey))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return "", errors.New("failed to decode PEM block containing private key")
	}
	privateKeyRSA, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	policy := &sign.Policy{
		Statements: []sign.Statement{
			{
				Resource: resource,
				Condition: sign.Condition{
					DateLessThan: &sign.AWSEpochTime{Time: time.Now().Add(1 * time.Hour)}, // Expiration
				},
			},
		},
	}

	cookieSigner := sign.NewCookieSigner(keyPairID, privateKeyRSA)
	signedCookies, err := cookieSigner.SignWithPolicy(policy)
	if err != nil {
		return "", fmt.Errorf("failed to sign cookies: %w", err)
	}

	cookies := ""
	for _, cookie := range signedCookies {
		cookies += fmt.Sprintf("%s=%s; ", cookie.Name, cookie.Value)
	}

	return cookies, nil
}
