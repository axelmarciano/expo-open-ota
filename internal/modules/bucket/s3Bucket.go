package bucket

import (
	"context"
	"errors"
	"expo-open-ota/internal/modules/types"
	"expo-open-ota/internal/services"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"strconv"
	"time"
)

type S3Bucket struct {
	BucketName string
}

func (b *S3Bucket) GetUpdates(environment string, runtimeVersion string) ([]types.Update, error) {
	if b.BucketName == "" {
		return nil, errors.New("BucketName not set")
	}
	s3Client, errS3 := services.GetS3Client()
	if errS3 != nil {
		return nil, errS3
	}
	prefix := environment + "/" + runtimeVersion + "/"
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(b.BucketName),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}
	resp, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("ListObjectsV2 error: %w", err)
	}
	var updates []types.Update
	for _, commonPrefix := range resp.CommonPrefixes {
		var updateId int64
		if _, err := fmt.Sscanf(*commonPrefix.Prefix, prefix+"%d/", &updateId); err == nil {
			updates = append(updates, types.Update{
				Environment:    environment,
				RuntimeVersion: runtimeVersion,
				UpdateId:       strconv.FormatInt(updateId, 10),
				CreatedAt:      time.Duration(updateId) * time.Millisecond,
			})
		}
	}
	return updates, nil
}

func (b *S3Bucket) GetFile(update types.Update, assetPath string) (types.BucketFile, error) {
	if b.BucketName == "" {
		return types.BucketFile{}, errors.New("BucketName not set")
	}
	filePath := update.Environment + "/" + update.RuntimeVersion + "/" + update.UpdateId + "/" + assetPath
	s3Client, errS3 := services.GetS3Client()
	if errS3 != nil {
		return types.BucketFile{}, errS3
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(filePath),
	}
	resp, err := s3Client.GetObject(context.TODO(), input)
	if err != nil {

		return types.BucketFile{}, fmt.Errorf("GetObject error: %w", err)
	}
	return types.BucketFile{
		Reader:    resp.Body,
		CreatedAt: *resp.LastModified,
	}, nil
}
