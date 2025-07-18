package xfer

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	HttpFileGetGetDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "capi_http_config_file_get_duration",
		Help:    "Duration of config file HTTP requests in seconds",
		Buckets: []float64{0.1, 1, 5},
	})
	S3FileGetGetDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "capi_s3_config_file_get_duration",
		Help:    "Duration of config file S3 requests in seconds",
		Buckets: []float64{0.1, 1, 5},
	})
	SftpFileGetGetDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "capi_sftp_config_file_get_duration",
		Help:    "Duration of config file SFTP requests in seconds",
		Buckets: []float64{0.1, 1, 5},
	})
)

func GetFileBytes(fileUrl string, certPath string, privateKeys map[string]string) ([]byte, error) {
	u, err := url.Parse(fileUrl)
	if err != nil {
		return nil, fmt.Errorf("cannot parse url %s: %s", fileUrl, err.Error())
	}

	var bytes []byte
	startTime := time.Now()
	if u.Scheme == UrlSchemeFile || len(u.Scheme) == 0 {
		bytes, err = os.ReadFile(fileUrl)
		if err != nil {
			return nil, err
		}
	} else if u.Scheme == UrlSchemeHttp || u.Scheme == UrlSchemeHttps {
		bytes, err = readHttpFile(fileUrl, u.Scheme, certPath)
		if err != nil {
			return nil, err
		}
		HttpFileGetGetDuration.Observe(float64(time.Since(startTime).Seconds()))
	} else if u.Scheme == UrlSchemeS3 {
		bytes, err = readS3File(fileUrl)
		if err != nil {
			return nil, err
		}
		S3FileGetGetDuration.Observe(float64(time.Since(startTime).Seconds()))
	} else if u.Scheme == UrlSchemeSftp {
		// When dealing with sftp, we download the *whole* file, then we read all of it
		dstFile, err := os.CreateTemp("", "capi")
		if err != nil {
			return nil, fmt.Errorf("cannot create temp file for %s: %s", fileUrl, err.Error())
		}

		// Download and schedule delete
		if err = DownloadSftpFile(fileUrl, privateKeys, dstFile); err != nil {
			dstFile.Close()
			return nil, err
		}
		dstFile.Close()
		defer os.Remove(dstFile.Name())

		// Read
		bytes, err = os.ReadFile(dstFile.Name())
		if err != nil {
			return nil, fmt.Errorf("cannot read from file %s downloaded from %s: %s", dstFile.Name(), fileUrl, err.Error())
		}
		SftpFileGetGetDuration.Observe(float64(time.Since(startTime).Seconds()))
	} else {
		return nil, fmt.Errorf("url scheme %s not supported: %s", u.Scheme, fileUrl)
	}

	return bytes, nil
}
