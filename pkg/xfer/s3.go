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
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetS3ReadCloser(uri string) (io.ReadCloser, error) {
	parsedUri, _ := url.Parse(uri)

	// Assuming ~/.aws/credentials:
	// [default]
	// aws_access_key_id=AKIA2TIMRSBWB77BEXWH
	// aws_secret_access_key=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
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

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(strings.TrimLeft(u.Path, "/")),
		Body:   f,
	})
	if err != nil {
		return err
	}

	return nil
}
