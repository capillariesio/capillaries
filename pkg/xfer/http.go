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

const UriSchemeFile string = "file"
const UriSchemeHttp string = "http"
const UriSchemeHttps string = "https"
const UriSchemeSftp string = "sftp"

func GetHttpReadCloser(uri string, scheme string, certDir string) (io.ReadCloser, error) {
	caCertPool := x509.NewCertPool()
	if scheme == UriSchemeHttps {
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
	t := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{}, RootCAs: caCertPool}}
	client := http.Client{Transport: t, Timeout: 30 * time.Second}

	resp, err := client.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot get %s: %s", uri, err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("cannot get %s, bad status: %s", uri, resp.Status)
	}

	return resp.Body, nil
}

func readHttpFile(uri string, scheme string, certDir string) ([]byte, error) {
	r, err := GetHttpReadCloser(uri, scheme, certDir)
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
