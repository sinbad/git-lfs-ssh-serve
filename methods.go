package main

import (
	"fmt"
	"github.com/github/git-lfs/lfs"
	"github.com/technoweenie/go-contentaddressable"
	"io"
	"os"
	"path/filepath"
)

func upload(req *lfs.JsonRequest, in io.Reader, out io.Writer, config *Config, path string) *lfs.JsonResponse {
	upreq := lfs.UploadRequest{}
	err := lfs.ExtractStructFromJsonRawMessage(req.Params, &upreq)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	// Build destination path
	filename, err := mediaPath(upreq.Oid, config)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Error determining media path. %v", err))
	}
	// contentaddressable fails hard when file already exists, let's be nicer
	startresult := lfs.UploadResponse{}
	_, staterr := os.Stat(filename)
	var mediaFile *contentaddressable.File
	if staterr != nil && os.IsNotExist(staterr) {
		startresult.OkToSend = true
		// Create file to write to - do this early to allow for fail states
		mediaFile, err = contentaddressable.NewFile(filename)
		if err != nil {
			return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Error opening media file buffer. %v", err))
		}
	}
	// Send start response immediately
	resp, err := lfs.NewJsonResponse(req.Id, startresult)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	err = sendResponse(resp, out)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	if !startresult.OkToSend {
		return nil
	}

	// Next from client should be byte stream of exactly the stated number of bytes
	n, err := io.CopyN(mediaFile, in, upreq.Size)
	if err == nil {
		err = mediaFile.Accept()
	}
	mediaFile.Close()
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Problem uploading data: %v", err.Error()))
	} else if n != upreq.Size {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Received wrong number of bytes %d (expected %d)", n, upreq.Size))
	}

	receivedresult := lfs.UploadCompleteResponse{}
	receivedresult.ReceivedOk = true
	resp, _ = lfs.NewJsonResponse(req.Id, receivedresult)

	return resp

}

func uploadCheck(req *lfs.JsonRequest, in io.Reader, out io.Writer, config *Config, path string) *lfs.JsonResponse {
	upreq := lfs.UploadRequest{}
	err := lfs.ExtractStructFromJsonRawMessage(req.Params, &upreq)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	// Build destination path
	filename, err := mediaPath(upreq.Oid, config)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Error determining media path. %v", err))
	}
	// contentaddressable fails hard when file already exists, let's be nicer
	startresult := lfs.UploadResponse{}
	_, staterr := os.Stat(filename)
	if staterr != nil && os.IsNotExist(staterr) {
		startresult.OkToSend = true
	}
	// Send start response immediately
	resp, err := lfs.NewJsonResponse(req.Id, startresult)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	return resp

}

func downloadCheck(req *lfs.JsonRequest, in io.Reader, out io.Writer, config *Config, path string) *lfs.JsonResponse {
	downreq := lfs.DownloadCheckRequest{}
	err := lfs.ExtractStructFromJsonRawMessage(req.Params, &downreq)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	filename, err := mediaPath(downreq.Oid, config)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Problem determining media path: %v", err))
	}
	result := lfs.DownloadCheckResponse{}
	s, err := os.Stat(filename)
	if err == nil {
		// file exists
		result.Size = s.Size()
	} else {
		result.Size = -1
	}
	resp, err := lfs.NewJsonResponse(req.Id, result)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	return resp
}
func download(req *lfs.JsonRequest, in io.Reader, out io.Writer, config *Config, path string) *lfs.JsonResponse {
	downreq := lfs.DownloadRequest{}
	err := lfs.ExtractStructFromJsonRawMessage(req.Params, &downreq)
	if err != nil {
		// Serve() copes with converting this to stderr rather than JSON response
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	filename, err := mediaPath(downreq.Oid, config)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Problem determining the media path: %v", err))
	}
	// check size
	s, err := os.Stat(filename)
	if err != nil {
		// file doesn't exist, this should not have been called
		return lfs.NewJsonErrorResponse(req.Id, "File doesn't exist")
	}
	if s.Size() != downreq.Size {
		// This won't work!
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("File sizes disagree (client: %d server: %d)", downreq.Size, s.Size()))
	}

	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	defer f.Close()

	n, err := io.Copy(out, f)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Error copying data to output: %v", err.Error()))
	}
	if n != s.Size() {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Amount of data copied disagrees (expected: %d actual: %d)", s.Size(), n))
	}
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Error copying data to output: %v", err.Error()))
	}
	if n != s.Size() {
		return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Amount of data copied disagrees (expected: %d actual: %d)", s.Size(), n))
	}

	// Don't return a response, only response is byte stream above except in error cases
	return nil
}

func batch(req *lfs.JsonRequest, in io.Reader, out io.Writer, config *Config, path string) *lfs.JsonResponse {
	batchreq := lfs.BatchRequest{}
	err := lfs.ExtractStructFromJsonRawMessage(req.Params, &batchreq)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	result := lfs.BatchResponse{}
	for _, o := range batchreq.Objects {
		filename, err := mediaPath(o.Oid, config)
		if err != nil {
			return lfs.NewJsonErrorResponse(req.Id, fmt.Sprintf("Problem determining the media path: %v", err))
		}
		resultObj := lfs.BatchResponseObject{Oid: o.Oid}
		s, err := os.Stat(filename)
		if err == nil {
			// file exists
			resultObj.Action = "download"
			resultObj.Size = s.Size()
		} else {
			resultObj.Action = "upload"
			resultObj.Size = o.Size
		}
		result.Results = append(result.Results, resultObj)
	}

	resp, err := lfs.NewJsonResponse(req.Id, result)
	if err != nil {
		return lfs.NewJsonErrorResponse(req.Id, err.Error())
	}
	return resp

}

// Store in the same structure as client, just under BasePath
func mediaPath(sha string, config *Config) (string, error) {
	path := filepath.Join(config.BasePath, sha[0:2], sha[2:4])
	if err := os.MkdirAll(path, 0744); err != nil {
		return "", fmt.Errorf("Error trying to create local media directory in '%s': %s", path, err)
	}
	return filepath.Join(path, sha), nil
}
