package xfer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetS3ReadCloser(uri string) (io.ReadCloser, error) {
	parsedUri, _ := url.Parse(uri)

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

	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(parsedUri.Host),
		Key:    aws.String(strings.TrimLeft(parsedUri.Path, "/")),
	})
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func readS3File(uri string) ([]byte, error) {
	r, err := GetS3ReadCloser(uri)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	bytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("cannot read body of %s, bad status: %s", uri, err.Error())
	}
	return bytes, nil
}

func UploadS3File(srcPath string, u *url.URL) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
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
