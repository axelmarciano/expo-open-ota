package services

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"log"
	"sync"
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
