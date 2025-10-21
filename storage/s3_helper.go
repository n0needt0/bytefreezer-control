package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n0needt0/go-goodies/log"
)

// S3Cleaner handles S3 bucket cleanup operations
type S3Cleaner struct {
	client *s3.Client
}

// NewS3Cleaner creates a new S3 cleaner with explicit credentials
// This function ONLY uses the provided credentials and does not fall back
// to the AWS default credential chain (env vars, shared config, IAM role, EC2 IMDS)
func NewS3Cleaner(ctx context.Context, accessKey, secretKey, region, endpoint string, useSSL bool) (*S3Cleaner, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("S3 access_key and secret_key are required for S3 cleanup operations")
	}

	// Create config with ONLY static credentials - no default credential chain fallback
	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		),
	}

	// Create S3 client
	var s3Client *s3.Client
	if endpoint != "" {
		// Custom endpoint (e.g., MinIO, LocalStack)
		// Construct full endpoint URL with protocol
		protocol := "http"
		if useSSL {
			protocol = "https"
		}
		fullEndpoint := fmt.Sprintf("%s://%s", protocol, endpoint)

		log.Infof("Creating S3 client with custom endpoint: %s (useSSL=%v)", fullEndpoint, useSSL)

		s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(fullEndpoint)
			o.UsePathStyle = true // Required for MinIO/LocalStack
		})
	} else {
		s3Client = s3.NewFromConfig(cfg)
	}

	return &S3Cleaner{
		client: s3Client,
	}, nil
}

// DeletePrefix deletes all objects under a specific prefix in a bucket
func (s *S3Cleaner) DeletePrefix(ctx context.Context, bucket, prefix string) error {
	log.Infof("Starting S3 cleanup for bucket=%s prefix=%s", bucket, prefix)

	// List all objects with the prefix
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	totalObjects := 0
	deletedObjects := 0

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects in bucket %s with prefix %s: %w", bucket, prefix, err)
		}

		if len(page.Contents) == 0 {
			log.Debugf("No objects found in bucket=%s prefix=%s", bucket, prefix)
			continue
		}

		totalObjects += len(page.Contents)

		// Delete objects individually for MinIO compatibility
		// MinIO requires Content-MD5 header for batch deletes, which AWS SDK v2 doesn't provide
		// Individual deletes are more compatible and avoid the MissingContentMD5 error
		for _, obj := range page.Contents {
			_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    obj.Key,
			})

			if err != nil {
				log.Warnf("Failed to delete object %s from bucket %s: %v", *obj.Key, bucket, err)
				// Continue with other objects even if one fails
				continue
			}

			deletedObjects++
		}

		log.Debugf("Deleted %d objects from bucket=%s prefix=%s", len(page.Contents), bucket, prefix)
	}

	log.Infof("Completed S3 cleanup for bucket=%s prefix=%s: found %d objects, deleted %d objects",
		bucket, prefix, totalObjects, deletedObjects)

	return nil
}

// CleanupDatasetStorage deletes intermediate S3 data for a dataset
// ONLY deletes intake and piper data - NEVER deletes packer output (customer data)
//
// ByteFreezer uses separate S3 buckets for each stage:
// - intake bucket: Raw data from bytefreezer-receiver (DELETED)
// - piper bucket: Processed data from bytefreezer-piper (DELETED)
// - packer bucket: Final parquet output (PRESERVED - customer data)
//
// The dataset configuration remains in the database for re-use
func (s *S3Cleaner) CleanupDatasetStorage(ctx context.Context, intakeBucket, piperBucket, tenantID, datasetID string) error {
	// Define the buckets and paths to delete (ONLY intermediate data, NOT final output)
	// Note: We use the same credentials but different bucket names
	cleanupTargets := []struct {
		name   string
		bucket string
		prefix string
	}{
		{"intake", intakeBucket, fmt.Sprintf("%s/%s/", tenantID, datasetID)},
		{"piper", piperBucket, fmt.Sprintf("%s/%s/", tenantID, datasetID)},
		// NOTE: packer output is NEVER deleted - it belongs to the customer
	}

	// Track errors but continue with all deletions
	var cleanupErrors []error

	for _, target := range cleanupTargets {
		log.Infof("Cleaning up %s bucket for dataset %s/%s", target.name, tenantID, datasetID)

		if err := s.DeletePrefix(ctx, target.bucket, target.prefix); err != nil {
			log.Errorf("Failed to cleanup %s bucket: %v", target.name, err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("%s bucket: %w", target.name, err))
		} else {
			log.Infof("Successfully cleaned up %s bucket", target.name)
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("S3 cleanup completed with %d errors: %v", len(cleanupErrors), cleanupErrors)
	}

	log.Infof("Dataset %s/%s S3 cleanup complete. Deleted intake and piper data. Packer output preserved for customer.", tenantID, datasetID)
	return nil
}

// GetS3ConfigFromDataset extracts S3 configuration from dataset config
// Falls back to environment variables if dataset config is incomplete
func GetS3ConfigFromDataset(dataset *Dataset) (bucket, region, endpoint, accessKey, secretKey string, useSSL bool, err error) {
	// Check destination config
	if dataset.Config.Destination.Connection.Bucket != "" {
		bucket = dataset.Config.Destination.Connection.Bucket
	}

	// Region: dataset config > env var > default
	if dataset.Config.Destination.Connection.Region != "" {
		region = dataset.Config.Destination.Connection.Region
	} else if envRegion := os.Getenv("AWS_REGION"); envRegion != "" {
		region = envRegion
	} else {
		region = "us-east-1"
	}

	// Endpoint: dataset config > env var
	if dataset.Config.Destination.Connection.Endpoint != "" {
		endpoint = dataset.Config.Destination.Connection.Endpoint
	} else {
		endpoint = os.Getenv("S3_ENDPOINT")
	}

	// SSL flag: dataset config > env var > default false (for MinIO)
	if dataset.Config.Destination.Connection.Credentials.AccessKey != "" {
		useSSL = dataset.Config.Destination.Connection.SSL
	} else if os.Getenv("S3_USE_SSL") == "true" {
		useSSL = true
	} else {
		useSSL = false
	}

	// Credentials: dataset config > environment variables
	if dataset.Config.Destination.Connection.Credentials.AccessKey != "" {
		accessKey = dataset.Config.Destination.Connection.Credentials.AccessKey
		secretKey = dataset.Config.Destination.Connection.Credentials.SecretKey
	} else {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	// For cleanup, we use fixed bucket names (intake, piper, packer)
	// So we don't require bucket to be set in dataset config
	// Just use "packer" as default for the bucket parameter
	if bucket == "" {
		bucket = "packer" // Default packer bucket name
	}

	return bucket, region, endpoint, accessKey, secretKey, useSSL, nil
}
