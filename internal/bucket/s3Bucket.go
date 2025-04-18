package bucket

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"expo-open-ota/internal/services"
	"expo-open-ota/internal/types"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type S3Bucket struct {
	BucketName string
}

func (b *S3Bucket) DeleteUpdateFolder(branch, runtimeVersion, updateId string) error {
	if b.BucketName == "" {
		return errors.New("BucketName not set")
	}

	s3Client, err := services.GetS3Client()
	if err != nil {
		return fmt.Errorf("error getting S3 client: %w", err)
	}

	prefix := fmt.Sprintf("%s/%s/%s/", branch, runtimeVersion, updateId)

	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(b.BucketName),
		Prefix: aws.String(prefix),
	}

	var objects []s3types.ObjectIdentifier

	paginator := s3.NewListObjectsV2Paginator(s3Client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, s3types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	if len(objects) == 0 {
		return nil
	}

	const batchSize = 1000
	for i := 0; i < len(objects); i += batchSize {
		end := i + batchSize
		if end > len(objects) {
			end = len(objects)
		}

		deleteInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(b.BucketName),
			Delete: &s3types.Delete{
				Objects: objects[i:end],
				Quiet:   aws.Bool(true),
			},
		}

		_, err := s3Client.DeleteObjects(context.TODO(), deleteInput)
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}
	}

	return nil
}

func (b *S3Bucket) GetRuntimeVersions(branch string) ([]RuntimeVersionWithStats, error) {
	if b.BucketName == "" {
		return nil, errors.New("BucketName not set")
	}
	s3Client, errS3 := services.GetS3Client()
	if errS3 != nil {
		return nil, errS3
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(b.BucketName),
		Prefix:    aws.String(branch + "/"),
		Delimiter: aws.String("/"),
	}
	resp, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("ListObjectsV2 error: %w", err)
	}

	var runtimeVersions []RuntimeVersionWithStats
	prefixLen := len(branch) + 1

	for _, commonPrefix := range resp.CommonPrefixes {
		runtimeVersion := (*commonPrefix.Prefix)[prefixLen : len(*commonPrefix.Prefix)-1]
		updatesPath := *commonPrefix.Prefix
		updateInput := &s3.ListObjectsV2Input{
			Bucket:    aws.String(b.BucketName),
			Prefix:    aws.String(updatesPath),
			Delimiter: aws.String("/"),
		}
		updateResp, err := s3Client.ListObjectsV2(context.TODO(), updateInput)
		if err != nil {
			return nil, fmt.Errorf("ListObjectsV2 error in updates: %w", err)
		}

		var updateTimestamps []int64
		for _, commonPrefix := range updateResp.CommonPrefixes {
			updateID := strings.TrimSuffix((*commonPrefix.Prefix)[len(updatesPath):], "/")
			timestamp, err := strconv.ParseInt(updateID, 10, 64)
			if err != nil {
				continue
			}
			updateTimestamps = append(updateTimestamps, timestamp)
		}

		if len(updateTimestamps) == 0 {
			continue
		}

		sort.Slice(updateTimestamps, func(i, j int) bool { return updateTimestamps[i] < updateTimestamps[j] })

		runtimeVersions = append(runtimeVersions, RuntimeVersionWithStats{
			RuntimeVersion:  runtimeVersion,
			CreatedAt:       time.UnixMilli(updateTimestamps[0]).UTC().Format(time.RFC3339),
			LastUpdatedAt:   time.UnixMilli(updateTimestamps[len(updateTimestamps)-1]).UTC().Format(time.RFC3339),
			NumberOfUpdates: len(updateTimestamps),
		})
	}

	return runtimeVersions, nil
}

func (b *S3Bucket) GetBranches() ([]string, error) {
	if b.BucketName == "" {
		return nil, errors.New("BucketName not set")
	}
	s3Client, errS3 := services.GetS3Client()
	if errS3 != nil {
		return nil, errS3
	}
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(b.BucketName),
		Delimiter: aws.String("/"),
	}
	resp, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("ListObjectsV2 error: %w", err)
	}
	var branches []string
	for _, commonPrefix := range resp.CommonPrefixes {
		prefix := *commonPrefix.Prefix
		branches = append(branches, prefix[:len(prefix)-1])
	}
	return branches, nil
}

func (b *S3Bucket) GetUpdates(branch string, runtimeVersion string) ([]types.Update, error) {
	if b.BucketName == "" {
		return nil, errors.New("BucketName not set")
	}
	s3Client, errS3 := services.GetS3Client()
	if errS3 != nil {
		return nil, errS3
	}
	prefix := branch + "/" + runtimeVersion + "/"
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
				Branch:         branch,
				RuntimeVersion: runtimeVersion,
				UpdateId:       strconv.FormatInt(updateId, 10),
				CreatedAt:      time.Duration(updateId) * time.Millisecond,
			})
		}
	}
	return updates, nil
}

func (b *S3Bucket) GetFile(update types.Update, assetPath string) (*types.BucketFile, error) {
	if b.BucketName == "" {
		return nil, errors.New("BucketName not set")
	}
	key := update.Branch + "/" + update.RuntimeVersion + "/" + update.UpdateId + "/" + assetPath

	s3Client, err := services.GetS3Client()
	if err != nil {
		return nil, err
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(key),
	}
	resp, err := s3Client.GetObject(context.TODO(), input)
	if err != nil {
		var noSuchKey *s3types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetObject error: %w", err)
	}

	return &types.BucketFile{
		Reader:    resp.Body,
		CreatedAt: *resp.LastModified,
	}, nil
}

func (b *S3Bucket) RequestUploadUrlForFileUpdate(branch string, runtimeVersion string, updateId string, fileName string) (string, error) {
	if b.BucketName == "" {
		return "", errors.New("BucketName not set")
	}

	s3Client, err := services.GetS3Client()
	if err != nil {
		return "", fmt.Errorf("error getting S3 client: %w", err)
	}

	presignClient := s3.NewPresignClient(s3Client)

	key := fmt.Sprintf("%s/%s/%s/%s", branch, runtimeVersion, updateId, fileName)

	input := &s3.PutObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(key),
	}

	presignResult, err := presignClient.PresignPutObject(context.TODO(), input, func(opt *s3.PresignOptions) {
		opt.Expires = 15 * time.Minute
	})
	if err != nil {
		return "", fmt.Errorf("error presigning URL: %w", err)
	}

	return presignResult.URL, nil
}

func (b *S3Bucket) UploadFileIntoUpdate(update types.Update, fileName string, file io.Reader) error {
	if b.BucketName == "" {
		return errors.New("BucketName not set")
	}
	s3Client, err := services.GetS3Client()
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s/%s/%s/%s", update.Branch, update.RuntimeVersion, update.UpdateId, fileName)
	input := &s3.PutObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(key),
		Body:   file,
	}
	_, err = s3Client.PutObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("PutObject error: %w", err)
	}
	return nil
}

func (b *S3Bucket) CreateUpdateFrom(previousUpdate *types.Update, newUpdateId string) (*types.Update, error) {
	if b.BucketName == "" {
		return nil, errors.New("BucketName not set")
	}
	if previousUpdate == nil {
		return nil, errors.New("previousUpdate is nil")
	}
	if previousUpdate.UpdateId == "" {
		return nil, errors.New("previousUpdate.UpdateId is empty")
	}
	if newUpdateId == "" {
		return nil, errors.New("newUpdateId is empty")
	}

	s3Client, err := services.GetS3Client()
	if err != nil {
		return nil, fmt.Errorf("error getting S3 client: %w", err)
	}

	sourcePrefix := fmt.Sprintf("%s/%s/%s/", previousUpdate.Branch, previousUpdate.RuntimeVersion, previousUpdate.UpdateId)
	targetPrefix := fmt.Sprintf("%s/%s/%s/", previousUpdate.Branch, previousUpdate.RuntimeVersion, newUpdateId)

	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.BucketName),
		Prefix: aws.String(sourcePrefix),
	})

	var wg sync.WaitGroup
	errChan := make(chan error, 16)
	sem := make(chan struct{}, runtime.NumCPU())

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, object := range page.Contents {
			key := *object.Key
			relPath := strings.TrimPrefix(key, sourcePrefix)

			if relPath == "update-metadata.json" || relPath == ".check" || strings.HasSuffix(relPath, "/") {
				continue
			}

			srcKey := key
			dstKey := targetPrefix + relPath

			wg.Add(1)
			go func(srcKey, dstKey string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				getObjOutput, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
					Bucket: aws.String(b.BucketName),
					Key:    aws.String(srcKey),
				})
				if err != nil {
					errChan <- fmt.Errorf("error getting object %s: %w", srcKey, err)
					return
				}
				defer getObjOutput.Body.Close()

				_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
					Bucket: aws.String(b.BucketName),
					Key:    aws.String(dstKey),
					Body:   getObjOutput.Body,
				})
				if err != nil {
					errChan <- fmt.Errorf("error putting object %s: %w", dstKey, err)
					return
				}
			}(srcKey, dstKey)
		}
	}

	wg.Wait()
	close(errChan)

	for e := range errChan {
		if e != nil {
			return nil, e
		}
	}

	updateId, err := strconv.ParseInt(newUpdateId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing update ID: %w", err)
	}
	return &types.Update{
		Branch:         previousUpdate.Branch,
		RuntimeVersion: previousUpdate.RuntimeVersion,
		UpdateId:       newUpdateId,
		CreatedAt:      time.Duration(updateId) * time.Millisecond,
	}, nil
}

func (b *S3Bucket) RetrieveMigrationHistory() ([]string, error) {
	if b.BucketName == "" {
		return nil, errors.New("BucketName not set")
	}
	ctx := context.TODO()
	s3Client, err := services.GetS3Client()
	if err != nil {
		return nil, err
	}
	const key = ".migrationhistory"
	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *s3types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: aws.String(b.BucketName),
				Key:    aws.String(key),
				Body:   bytes.NewReader([]byte{}),
			})
			if err != nil {
				return nil, fmt.Errorf("PutObject error: %w", err)
			}
			return []string{}, nil
		}
		return nil, fmt.Errorf("GetObject error: %w", err)
	}
	defer resp.Body.Close()

	var history []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		history = append(history, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}
	return history, nil
}

func (b *S3Bucket) ApplyMigration(migrationId string) error {
	if b.BucketName == "" {
		return errors.New("BucketName not set")
	}

	migrationHistory, err := b.RetrieveMigrationHistory()
	if err != nil {
		return fmt.Errorf("RetrieveMigrationHistory error: %w", err)
	}
	isAlreadyApplied := false
	for _, id := range migrationHistory {
		if id == migrationId {
			isAlreadyApplied = true
			break
		}
	}
	if isAlreadyApplied {
		return nil
	}

	s3Client, errS3 := services.GetS3Client()
	if errS3 != nil {
		return errS3
	}

	var currentContent []byte
	obj, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(".migrationhistory"),
	})
	if err == nil {
		defer obj.Body.Close()
		currentContent, _ = io.ReadAll(obj.Body)
	}

	newContent := append(currentContent, []byte(migrationId+"\n")...)

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(".migrationhistory"),
		Body:   bytes.NewReader(newContent),
	})
	if err != nil {
		return fmt.Errorf("PutObject error: %w", err)
	}

	return nil
}

func (b *S3Bucket) RemoveMigrationFromHistory(migrationId string) error {
	if b.BucketName == "" {
		return errors.New("BucketName not set")
	}

	migrationHistory, err := b.RetrieveMigrationHistory()
	if err != nil {
		return fmt.Errorf("RetrieveMigrationHistory error: %w", err)
	}

	hasMigration := false
	for _, id := range migrationHistory {
		if id == migrationId {
			hasMigration = true
			break
		}
	}
	if !hasMigration {
		return nil
	}

	var newContent []byte
	for _, id := range migrationHistory {
		if id != migrationId {
			newContent = append(newContent, []byte(id+"\n")...)
		}
	}

	s3Client, errS3 := services.GetS3Client()
	if errS3 != nil {
		return errS3
	}

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(b.BucketName),
		Key:    aws.String(".migrationhistory"),
		Body:   bytes.NewReader(newContent),
	})
	if err != nil {
		return fmt.Errorf("PutObject error: %w", err)
	}

	return nil
}
