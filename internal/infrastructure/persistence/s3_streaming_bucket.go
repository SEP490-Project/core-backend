package persistence

import (
	"context"
	"core-backend/config"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

type S3StreamingBucket struct {
	Client           *s3.Client
	BucketName       string
	Region           string
	CloudfrontDomain string
}

func InitS3StreamingBucket() *S3StreamingBucket {
	zap.L().Info("Initializing S3 connection")

	s3Cfg := config.GetAppConfig().S3StreamingBucket
	zap.L().Debug("S3 configuration loaded",
		zap.String("bucket", s3Cfg.BucketName),
		zap.String("region", s3Cfg.Region),
		zap.String("cloudfront_domain", s3Cfg.CloudfrontDomain),
		zap.Bool("access_key", s3Cfg.AccessKey != ""),
		zap.Bool("secret_key", s3Cfg.SecretKey != ""),
	)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(s3Cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3Cfg.AccessKey, s3Cfg.SecretKey, ""),
		),
	)
	if err != nil {
		zap.L().Panic("Failed to initialize AWS SDK", zap.Error(err))
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if s3Cfg.Endpoint != "" && strings.Contains(s3Cfg.Endpoint, "localhost") {
			o.BaseEndpoint = &s3Cfg.Endpoint
			o.UsePathStyle = true
		}
	})

	// verify bucket
	_, err = client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(s3Cfg.BucketName),
	})
	if err != nil {
		zap.L().Panic("S3 bucket verification failed", zap.Error(err))
	}

	zap.L().Info("S3 connection verified",
		zap.String("bucket", s3Cfg.BucketName),
		zap.String("region", s3Cfg.Region),
	)

	return &S3StreamingBucket{
		Client:           client,
		BucketName:       s3Cfg.BucketName,
		Region:           s3Cfg.Region,
		CloudfrontDomain: s3Cfg.CloudfrontDomain,
	}
}
