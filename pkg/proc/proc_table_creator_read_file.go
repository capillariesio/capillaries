package proc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
)

func checkRunReadFileForBatchSanity(node *sc.ScriptNodeDef, srcFileIdx int) error {
	if !node.HasFileReader() {
		return errors.New("node does not have file reader")
	}
	if !node.HasTableCreator() {
		return errors.New("node does not have table creator")
	}

	if srcFileIdx < 0 || srcFileIdx >= len(node.FileReader.SrcFileUrls) {
		return fmt.Errorf("cannot find file to read: asked to read src file with index %d while there are only %d source files available", srcFileIdx, len(node.FileReader.SrcFileUrls))
	}
	return nil
}

func runReadFileForBatch(envConfig *env.EnvConfig, logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, srcFileIdx int) (BatchStats, error) {
	logger.PushF("proc.runReadFileForBatch")
	defer logger.PopF()

	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0}

	node := pCtx.CurrentScriptNode

	if err := checkRunReadFileForBatchSanity(node, srcFileIdx); err != nil {
		return bs, err
	}

	filePath := node.FileReader.SrcFileUrls[srcFileIdx]

	u, err := url.Parse(filePath)
	if err != nil {
		return bs, fmt.Errorf("cannot parse file url %s: %s", filePath, err.Error())
	}

	bs.Src = filePath
	bs.Dst = node.TableCreator.Name + cql.RunIdSuffix(pCtx.Msg.RunId)

	var localSrcFile *os.File
	var fileReader io.Reader
	var fileReadSeeker io.ReadSeeker
	if u.Scheme == xfer.UrlSchemeFile || len(u.Scheme) == 0 {
		localSrcFile, err = os.Open(filePath)
		if err != nil {
			return bs, err
		}
		defer localSrcFile.Close()
		fileReader = bufio.NewReader(localSrcFile)
		fileReadSeeker = localSrcFile
	} else if u.Scheme == xfer.UrlSchemeHttp || u.Scheme == xfer.UrlSchemeHttps || u.Scheme == xfer.UrlSchemeS3 {
		var readCloser io.ReadCloser
		switch u.Scheme {
		case xfer.UrlSchemeHttp, xfer.UrlSchemeHttps:
			readCloser, err = xfer.GetHttpReadCloser(filePath, u.Scheme, envConfig.CaPath)
			if err != nil {
				return bs, fmt.Errorf("cannot open http file %s: %s", filePath, err.Error())
			}
		case xfer.UrlSchemeS3:
			readCloser, err = xfer.GetS3ReadCloser(filePath)
			if err != nil {
				return bs, fmt.Errorf("cannot open s3 file %s: %s", filePath, err.Error())
			}
		default:
			return bs, fmt.Errorf("cannot open file %s: unknown url scheme", filePath)
		}
		defer readCloser.Close()

		// If this is a parquet file, download it and then open so we have fileReadSeeker
		if node.FileReader.ReaderFileType == sc.ReaderFileTypeParquet {
			dstFile, err := os.CreateTemp("", "capi")
			if err != nil {
				return bs, fmt.Errorf("cannot create temp file for %s: %s", filePath, err.Error())
			}

			if _, err := io.Copy(dstFile, readCloser); err != nil {
				dstFile.Close()
				return bs, fmt.Errorf("cannot save http file %s to temp file %s: %s", filePath, dstFile.Name(), err.Error())
			}

			logger.Info("downloaded http file %s to %s", filePath, dstFile.Name())
			dstFile.Close()
			defer os.Remove(dstFile.Name())

			localSrcFile, err = os.Open(dstFile.Name())
			if err != nil {
				return bs, fmt.Errorf("cannot read from file %s downloaded from %s: %s", dstFile.Name(), filePath, err.Error())
			}
			defer localSrcFile.Close()
			fileReadSeeker = localSrcFile
		} else {
			// Just read from the net
			fileReader = readCloser
			defer readCloser.Close()
		}
	} else if u.Scheme == xfer.UrlSchemeSftp {
		// When dealing with sftp, we download the *whole* file, instead of providing a reader
		dstFile, err := os.CreateTemp("", "capi")
		if err != nil {
			return bs, fmt.Errorf("cannot create temp file for %s: %s", filePath, err.Error())
		}

		// Download and schedule delete
		if err = xfer.DownloadSftpFile(filePath, envConfig.PrivateKeys, dstFile); err != nil {
			dstFile.Close()
			return bs, err
		}
		logger.Info("downloaded sftp file %s to %s", filePath, dstFile.Name())
		dstFile.Close()
		defer os.Remove(dstFile.Name())

		// Create a reader for the temp file
		localSrcFile, err = os.Open(dstFile.Name())
		if err != nil {
			return bs, fmt.Errorf("cannot read from file %s downloaded from %s: %s", dstFile.Name(), filePath, err.Error())
		}
		defer localSrcFile.Close()
		fileReader = bufio.NewReader(localSrcFile)
		fileReadSeeker = localSrcFile
	} else {
		return bs, fmt.Errorf("l scheme %s not supported: %s", u.Scheme, filePath)
	}

	switch node.FileReader.ReaderFileType {
	case sc.ReaderFileTypeCsv:
		return readCsv(envConfig, logger, pCtx, totalStartTime, filePath, fileReader)
	case sc.ReaderFileTypeParquet:
		return readParquet(envConfig, logger, pCtx, totalStartTime, filePath, fileReadSeeker)
	default:
		return BatchStats{RowsRead: 0, RowsWritten: 0}, fmt.Errorf("unknown reader file type: %d", node.FileReader.ReaderFileType)
	}
}
