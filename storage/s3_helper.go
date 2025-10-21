package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/n0needt0/go-goodies/log"
)

// S3Cleaner handles S3 bucket cleanup operations
type S3Cleaner struct {
	client *s3.Client
}

// NewS3Cleaner creates a new S3 cleaner with credentials from dataset config
func NewS3Cleaner(ctx context.Context, accessKey, secretKey, region, endpoint string, useSSL bool) (*S3Cleaner, error) {
	// If no explicit credentials provided, try environment or default config
	var cfg aws.Config
	var err error

	if accessKey != "" && secretKey != "" {
		// Use explicit credentials
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				accessKey,
				secretKey,
				"",
			)),
		)
	} else {
		// Use default credential chain (env vars, shared config, IAM role, etc.)
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
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

		// Delete objects in batches (max 1000 per request)
		batchSize := 1000
		for i := 0; i < len(page.Contents); i += batchSize {
			end := i + batchSize
			if end > len(page.Contents) {
				end = len(page.Contents)
			}

			batch := page.Contents[i:end]
			objectIdentifiers := make([]types.ObjectIdentifier, 0, len(batch))
			for _, obj := range batch {
				objectIdentifiers = append(objectIdentifiers, types.ObjectIdentifier{
					Key: obj.Key,
				})
			}

			// Delete batch
			_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &types.Delete{
					Objects: objectIdentifiers,
					Quiet:   aws.Bool(true),
				},
			})

			if err != nil {
				return fmt.Errorf("failed to delete batch of objects from bucket %s: %w", bucket, err)
			}

			deletedObjects += len(objectIdentifiers)
			log.Debugf("Deleted batch of %d objects from bucket=%s prefix=%s", len(objectIdentifiers), bucket, prefix)
		}
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
func (s *S3Cleaner) CleanupDatasetStorage(ctx context.Context, packerBucket, tenantID, datasetID string) error {
	// Define the buckets and paths to delete (ONLY intermediate data, NOT final output)
	// Note: We use the same credentials but different bucket names
	cleanupTargets := []struct {
		name   string
		bucket string
		prefix string
	}{
		{"intake", "intake", fmt.Sprintf("%s/%s/", tenantID, datasetID)},
		{"piper", "piper", fmt.Sprintf("%s/%s/", tenantID, datasetID)},
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

	log.Infof("Dataset %s/%s S3 cleanup complete. Deleted intake and piper data. Preserved packer output (bucket=%s) for customer.", tenantID, datasetID, packerBucket)
	return nil
}

// GetS3ConfigFromDataset extracts S3 configuration from dataset config
func GetS3ConfigFromDataset(dataset *Dataset) (bucket, region, endpoint, accessKey, secretKey string, useSSL bool, err error) {
	// Check destination config
	if dataset.Config.Destination.Connection.Bucket != "" {
		bucket = dataset.Config.Destination.Connection.Bucket
	}

	if dataset.Config.Destination.Connection.Region != "" {
		region = dataset.Config.Destination.Connection.Region
	} else {
		// Default region if not specified
		region = "us-east-1"
	}

	if dataset.Config.Destination.Connection.Endpoint != "" {
		endpoint = dataset.Config.Destination.Connection.Endpoint
	}

	// Get SSL flag (defaults to false for MinIO compatibility)
	useSSL = dataset.Config.Destination.Connection.SSL

	// Get credentials from dataset config or environment
	if dataset.Config.Destination.Connection.Credentials.AccessKey != "" {
		accessKey = dataset.Config.Destination.Connection.Credentials.AccessKey
		secretKey = dataset.Config.Destination.Connection.Credentials.SecretKey
	} else {
		// Try environment variables
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	if bucket == "" {
		return "", "", "", "", "", false, fmt.Errorf("S3 bucket not configured in dataset")
	}

	return bucket, region, endpoint, accessKey, secretKey, useSSL, nil
}
