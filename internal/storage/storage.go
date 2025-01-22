package storage

import (
	"fmt"
	"mime/multipart"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func UploadFileToS3(file multipart.File, folder string, fileHeader *multipart.FileHeader, bucketName, awsRegion, awsAccessKey, awsSecretKey string) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	})

	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %v", err)
	}

	svc := s3.New(sess)


	fileName := fileHeader.Filename

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
		Body:   file,
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s", bucketName, awsRegion, folder, fileName)
	return fileURL, nil
}