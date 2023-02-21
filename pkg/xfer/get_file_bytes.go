package xfer

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
)

func GetFileBytes(uri string, certPath string, privateKeys map[string]string) ([]byte, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot parse uri %s: %s", uri, err.Error())
	}

	var bytes []byte
	if u.Scheme == UriSchemeFile || len(u.Scheme) == 0 {
		bytes, err = ioutil.ReadFile(uri)
	} else if u.Scheme == UriSchemeHttp || u.Scheme == UriSchemeHttps {
		bytes, err = readHttpFile(uri, u.Scheme, certPath)
	} else if u.Scheme == UriSchemeSftp {
		// When dealing with sftp, we download the *whole* file, then we read all of it
		dstFile, err := os.CreateTemp("", "capi")
		if err != nil {
			return nil, fmt.Errorf("cannot creeate temp file for %s: %s", uri, err.Error())
		}

		// Download and schedule delete
		if err = DownloadSftpFile(uri, privateKeys, dstFile); err != nil {
			dstFile.Close()
			return nil, err
		}
		dstFile.Close()
		defer os.Remove(dstFile.Name())

		// Read
		bytes, err = ioutil.ReadFile(dstFile.Name())
		if err != nil {
			err = fmt.Errorf("cannot read from file %s downloaded from %s: %s", dstFile.Name(), uri, err.Error())
		}
	} else {
		return nil, fmt.Errorf("uri scheme %s not supported: %s", u.Scheme, uri)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot read input from %s: %s", uri, err.Error())
	}

	return bytes, nil
}
