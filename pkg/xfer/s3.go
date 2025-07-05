package xfer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetS3ReadCloser(fileUrl string) (io.ReadCloser, error) {
	parsedUrl, _ := url.Parse(fileUrl)

	// Assuming ~/.aws/credentials:
	// [default]
	// aws_access_key_id=AKIA...
	// aws_secret_access_key=...
	// ~/.aws/config
	// [default]
	// region=us-east-1
	// output=json

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	// client := s3.New(s3.Options{
	// 	Region:      "us-east-1",
	// 	Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("AK...", "...", "")),
	// })

	// This may throw intermittent error:
	// operation error S3: GetObject, get identity: get credentials: failed to refresh cached credentials, no EC2 IMDS role found, operation error ec2imds: GetMetadata, canceled, context deadline exceeded
	maxRetries := 5
	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(parsedUrl.Host),
			Key:    aws.String(strings.TrimLeft(parsedUrl.Path, "/")),
		})
		if err == nil {
			return resp.Body, nil
		}
		if strings.Contains(err.Error(), "failed to refresh cached credentials, no EC2 IMDS role found, operation error ec2imds: GetMetadata, canceled, context deadline exceeded") {
			if retryCount < maxRetries-1 {
				time.Sleep(1 * time.Second)
			}
		} else {
			return nil, err
		}
	}

	return nil, fmt.Errorf("cannot download S3 file because of the EC2 IMDS error after %d attempts, giving up", maxRetries-1)
}

func readS3File(fileUrl string) ([]byte, error) {
	r, err := GetS3ReadCloser(fileUrl)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	bytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("cannot read body of %s, bad status: %s", fileUrl, err.Error())
	}
	return bytes, nil
}

func UploadS3File(srcPath string, u *url.URL) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open s3 file to upload %s: %s", srcPath, err.Error())
	}
	if f == nil {
		return fmt.Errorf("cannot open s3 file to upload %s: unknown error", srcPath)
	}
	defer f.Close()

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	uploader := manager.NewUploader(client)

	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(strings.TrimLeft(u.Path, "/")),
		Body:   f,
	})
	if err != nil {
		return err
	}

	return nil
}

type S3ConfigStatus struct {
	AccessKeyId     bool
	SecretAccessKey bool
	Region          string
	Err             string
}

func (st *S3ConfigStatus) String() string {
	return fmt.Sprintf("AccessKeyID:%t,SecretAccessKey:%t,Region:%s,Err:%s",
		st.AccessKeyId, st.SecretAccessKey, st.Region, st.Err)
}

func GetS3ConfigStatus(ctx context.Context) *S3ConfigStatus {
	st := S3ConfigStatus{}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		st.Err = fmt.Sprintf("cannot load s3 default config: %s", err.Error())
		return &st
	}

	st.Region = cfg.Region

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		st.Err = fmt.Sprintf("cannot retrieve s3 credentials: %s", err.Error())
		return &st
	}

	st.AccessKeyId = len(creds.AccessKeyID) > 0
	st.SecretAccessKey = len(creds.SecretAccessKey) > 0

	return &st
}
