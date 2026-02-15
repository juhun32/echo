package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	s3Client     *s3.Client
	s3BucketName string

	s3Uploading      bool
	lastS3UploadAt   time.Time
	hasLastS3Upload  bool
	lastS3DownloadAt time.Time
	hasLastS3Sync    bool
)

const cacheObjectKey = "cache.json"

func setUploading(uploading bool) {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	s3Uploading = uploading
}

func markS3UploadCompleted() {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	lastS3UploadAt = time.Now()
	hasLastS3Upload = true
}

func markS3DownloadCompleted() {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	lastS3DownloadAt = time.Now()
	hasLastS3Sync = true
}

func initS3Client() error {
	s3BucketName = strings.TrimSpace(os.Getenv("S3_BUCKET_NAME"))
	if s3BucketName == "" {
		return errors.New("S3_BUCKET_NAME is required")
	}

	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if region == "" {
		return errors.New("AWS_REGION is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	s3Client = s3.NewFromConfig(cfg)
	return nil
}

func downloadAndMergeFromS3() {
	if s3Client == nil || s3BucketName == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(cacheObjectKey),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") {
			log.Println("S3 cache.json not found; starting with empty cache")
			return
		}
		log.Printf("S3 download failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Read S3 cache body failed: %v", err)
		return
	}

	if len(body) == 0 {
		return
	}

	var remoteEntries []VectorEntry
	if err := json.Unmarshal(body, &remoteEntries); err != nil {
		log.Printf("Decode S3 cache.json failed: %v", err)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	existingByQuestion := make(map[string]struct{}, len(MockVectorDB))
	for _, entry := range MockVectorDB {
		questionKey := strings.TrimSpace(entry.Question)
		if questionKey != "" {
			existingByQuestion[questionKey] = struct{}{}
		}
	}

	newEntries := 0
	for _, entry := range remoteEntries {
		questionKey := strings.TrimSpace(entry.Question)
		if questionKey == "" {
			continue
		}
		if _, exists := existingByQuestion[questionKey]; exists {
			continue
		}
		entry.Source = cacheSourceS3
		MockVectorDB = append(MockVectorDB, entry)
		existingByQuestion[questionKey] = struct{}{}
		newEntries++
	}

	markS3DownloadCompleted()
	log.Printf("Synced: %d new entries found.", newEntries)
}

func uploadToS3() {
	if s3Client == nil || s3BucketName == "" {
		return
	}

	setUploading(true)
	defer setUploading(false)

	dbMutex.RLock()
	payload := make([]VectorEntry, len(MockVectorDB))
	copy(payload, MockVectorDB)
	dbMutex.RUnlock()

	jsonBody, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("Marshal cache for S3 failed: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s3BucketName),
		Key:         aws.String(cacheObjectKey),
		Body:        bytes.NewReader(jsonBody),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		log.Printf("S3 upload failed: %v", err)
		return
	}

	markS3UploadCompleted()
}

func startBackgroundSync() {
	ticker := time.NewTicker(300 * time.Second)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			downloadAndMergeFromS3()

			dbMutex.RLock()
			hasData := len(MockVectorDB) > 0
			dbMutex.RUnlock()

			if hasData {
				fmt.Println("Batching: Uploading memory to S3...")
				uploadToS3()
			}
		}
	}()
}
