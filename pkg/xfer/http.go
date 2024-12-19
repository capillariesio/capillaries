package xfer

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

const UrlSchemeFile string = "file"
const UrlSchemeHttp string = "http"
const UrlSchemeHttps string = "https"
const UrlSchemeSftp string = "sftp"
const UrlSchemeS3 string = "s3"

func GetHttpReadCloser(fileUrl string, scheme string, certDir string) (io.ReadCloser, error) {
	var caCertPool *x509.CertPool
	// tls.Config doc: If RootCAs is nil, TLS uses the host's root CA set.
	if certDir != "" {
		caCertPool = x509.NewCertPool()
		if scheme == UrlSchemeHttps {
			files, err := os.ReadDir(certDir)
			if err != nil {
				return nil, fmt.Errorf("cannot read ca dir with PEM certs %s: %s", certDir, err.Error())
			}

			for _, f := range files {
				caCert, err := os.ReadFile(path.Join(certDir, f.Name()))
				if err != nil {
					return nil, fmt.Errorf("cannot read PEM cert %s: %s", f.Name(), err.Error())
				}
				caCertPool.AppendCertsFromPEM(caCert)
			}
		}
	}
	t := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{}, RootCAs: caCertPool}}
	client := http.Client{Transport: t, Timeout: 30 * time.Second}

	resp, err := client.Get(fileUrl)
	if err != nil {
		return nil, fmt.Errorf("cannot get %s: %s", fileUrl, err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("cannot get %s, bad status: %s", fileUrl, resp.Status)
	}

	return resp.Body, nil
}

func readHttpFile(fileUrl string, scheme string, certDir string) ([]byte, error) {
	r, err := GetHttpReadCloser(fileUrl, scheme, certDir)
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
