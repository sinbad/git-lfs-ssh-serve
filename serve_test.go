package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/github/git-lfs/lfs"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmizerany/assert"
)

// Here we use a real client SSH context to talk to the real Serve() function
// However a Pipe is used to connect the two, no real SSH at this point
func TestServe(t *testing.T) {

	testcontentsz := int64(634)

	config := NewConfig()
	config.BasePath = filepath.Join(os.TempDir(), "git-lfs-serve-test")
	os.MkdirAll(config.BasePath, 0755)
	repopath := "test/repo"

	testcontent := make([]byte, testcontentsz)
	// put something interesting in it so we can detect it at each end
	testcontent[0] = '2'
	testcontent[1] = 'Z'
	testcontent[2] = '>'
	testcontent[3] = 'Q'
	testcontent[testcontentsz-1] = '#'
	testcontent[testcontentsz-2] = 'y'
	testcontent[testcontentsz-3] = 'L'
	testcontent[testcontentsz-4] = 'A'

	// Defer cleanup
	defer os.RemoveAll(config.BasePath)

	hasher := sha256.New()
	inbuf := bytes.NewReader(testcontent)
	io.Copy(hasher, inbuf)
	testoid := hex.EncodeToString(hasher.Sum(nil))

	cli, srv := net.Pipe()
	var outerr bytes.Buffer

	// 'Serve' is the real server function, usually connected to stdin/stdout but to pipe for test
	go Serve(srv, srv, &outerr, config, repopath)
	defer cli.Close()

	ctx := lfs.NewManualSSHApiContext(cli, cli)

	rdr := bytes.NewReader(testcontent)
	obj, wrerr := ctx.UploadCheck(testoid, int64(len(testcontent)))
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)
	assert.NotEqual(t, (*lfs.ObjectResource)(nil), wrerr)
	wrerr = ctx.UploadObject(obj, rdr)
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)
	assert.Equal(t, 0, rdr.Len()) // server should have read all bytes
	uploadDestPath, _ := mediaPath(testoid, config)
	s, err := os.Stat(uploadDestPath)
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(len(testcontent)), s.Size())

	// Prove that it fails safely when trying to upload duplicate content
	rdr = bytes.NewReader(testcontent)
	obj, wrerr = ctx.UploadCheck(testoid, int64(len(testcontent)))
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)
	assert.Equal(t, (*lfs.ObjectResource)(nil), obj)

	// Now try to download same data
	var dlbuf bytes.Buffer
	dlrdr, sz, wrerr := ctx.Download(testoid)
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)
	assert.Equal(t, testcontentsz, sz)
	_, err = io.CopyN(&dlbuf, dlrdr, sz)
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)

	downloadedbytes := dlbuf.Bytes()
	assert.Equal(t, testcontent, downloadedbytes)

	// Now separate DownloadCheck/DownloadObject
	dlbuf.Reset()
	obj, wrerr = ctx.DownloadCheck(testoid)
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)
	assert.NotEqual(t, (*lfs.ObjectResource)(nil), obj)
	assert.Equal(t, testoid, obj.Oid)
	assert.Equal(t, testcontentsz, obj.Size)
	assert.Equal(t, true, obj.CanDownload())
	assert.Equal(t, false, obj.CanUpload())

	dlrdr, sz, wrerr = ctx.DownloadObject(obj)
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)
	assert.Equal(t, testcontentsz, sz)
	_, err = io.CopyN(&dlbuf, dlrdr, sz)
	assert.Equal(t, (*lfs.WrappedError)(nil), wrerr)

	downloadedbytes = dlbuf.Bytes()
	assert.Equal(t, testcontent, downloadedbytes)

	// Now test safe fail state with DownloadCheck
	obj, wrerr = ctx.DownloadCheck("99999999999999999999999999999999999")
	assert.Equal(t, (*lfs.ObjectResource)(nil), obj)
	assert.NotEqual(t, (*lfs.WrappedError)(nil), wrerr)
	ctx.Close()

}
