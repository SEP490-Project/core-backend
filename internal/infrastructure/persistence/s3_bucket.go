package persistence

import (
	"context"
	"core-backend/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

type S3Bucket struct {
	Client     *s3.Client
	BucketName string
	Region     string
}

func InitS3() *S3Bucket {
	zap.L().Info("Initializing S3 connection")

	s3Cfg := config.GetAppConfig().S3Bucket
	zap.L().Debug("S3 configuration loaded",
		zap.String("bucket", s3Cfg.BucketName),
		zap.String("region", s3Cfg.Region),
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

	client := s3.NewFromConfig(awsCfg)

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

	return &S3Bucket{
		Client:     client,
		BucketName: s3Cfg.BucketName,
		Region:     s3Cfg.Region,
	}
}
