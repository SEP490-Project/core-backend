package third_party_repository

import (
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/infrastructure/persistence"
)

type ThirdPartyStorageRegistry struct {
	S3Storage       irepository_third_party.S3Storage          //File storage
	S3StreamStorage irepository_third_party.S3StreamingStorage //Video storage
}

func NewThirdPartyStorageRegistry(
	s3Bucket *persistence.S3Bucket, s3StreamBucket *persistence.S3StreamingBucket) *ThirdPartyStorageRegistry {
	return &ThirdPartyStorageRegistry{
		S3Storage:       NewS3Storage(s3Bucket),
		S3StreamStorage: NewS3StreamingStorage(s3StreamBucket),
	}
}
