package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/sinbad/git-lfs-ssh-serve/Godeps/_workspace/src/github.com/github/git-lfs/lfs"
	"io"
	"net"
	"os"
	"path/filepath"

	. "github.com/sinbad/git-lfs-ssh-serve/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/sinbad/git-lfs-ssh-serve/Godeps/_workspace/src/github.com/onsi/gomega"
)

// Here we use a real client SSH context to talk to the real Serve() function
// However a Pipe is used to connect the two, no real SSH at this point
var _ = Describe("Server tests", func() {

	var testcontentsz int64
	var config *Config
	var repopath string
	var testcontent []byte
	var testoid string

	BeforeEach(func() {

		testcontentsz = int64(634)
		config = NewConfig()
		config.BasePath = filepath.Join(os.TempDir(), "git-lfs-serve-test")
		os.MkdirAll(config.BasePath, 0755)
		repopath = "test/repo"

		testcontent = make([]byte, testcontentsz)
		// put something interesting in it so we can detect it at each end
		testcontent[0] = '2'
		testcontent[1] = 'Z'
		testcontent[2] = '>'
		testcontent[3] = 'Q'
		testcontent[testcontentsz-1] = '#'
		testcontent[testcontentsz-2] = 'y'
		testcontent[testcontentsz-3] = 'L'
		testcontent[testcontentsz-4] = 'A'

		hasher := sha256.New()
		inbuf := bytes.NewReader(testcontent)
		io.Copy(hasher, inbuf)
		testoid = hex.EncodeToString(hasher.Sum(nil))
	})
	AfterEach(func() {
		os.RemoveAll(config.BasePath)
	})

	It("Fulfils core server API", func() {
		cli, srv := net.Pipe()
		var outerr bytes.Buffer

		// 'Serve' is the real server function, usually connected to stdin/stdout but to pipe for test
		go Serve(srv, srv, &outerr, config, repopath)
		defer cli.Close()

		ctx := lfs.NewManualSSHApiContext(cli, cli)

		rdr := bytes.NewReader(testcontent)
		obj, wrerr := ctx.UploadCheck(testoid, int64(len(testcontent)))
		Expect(wrerr).To(BeNil(), "Should be no error on UploadCheck")
		Expect(obj).ToNot(BeNil(), "Should return valid resource")
		wrerr = ctx.UploadObject(obj, rdr)
		Expect(wrerr).To(BeNil(), "Should be no error on UploadObject")
		Expect(rdr.Len()).To(BeZero(), "Server should have read all the bytes")
		uploadDestPath, _ := mediaPath(testoid, config)
		s, err := os.Stat(uploadDestPath)
		Expect(err).To(BeNil(), "Destination file should exist")
		Expect(s.Size()).To(BeEquivalentTo(testcontentsz), "Destination file should be the correct length")

		// Prove that it fails safely when trying to upload duplicate content
		rdr = bytes.NewReader(testcontent)
		obj, wrerr = ctx.UploadCheck(testoid, int64(len(testcontent)))
		Expect(wrerr).To(BeNil(), "Should not report error when UploadCheck on existing file")
		Expect(obj).To(BeNil(), "Should return a nil resource when UploadCheck on an existing file")

		// Now try to download same data
		var dlbuf bytes.Buffer
		dlrdr, sz, wrerr := ctx.Download(testoid)
		Expect(wrerr).To(BeNil(), "Should not report error on Download")
		Expect(sz).To(BeEquivalentTo(testcontentsz), "Download should report the correct size")
		_, err = io.CopyN(&dlbuf, dlrdr, sz)
		Expect(err).To(BeNil(), "Should copy from the download stream successfully")

		downloadedbytes := dlbuf.Bytes()
		Expect(downloadedbytes).To(Equal(testcontent), "Downloaded bytes should be identical to original content")

		// Now separate DownloadCheck/DownloadObject
		dlbuf.Reset()
		obj, wrerr = ctx.DownloadCheck(testoid)
		Expect(wrerr).To(BeNil(), "Should not report error in DownloadCheck")
		Expect(obj).ToNot(BeNil(), "DownloadCheck should return a valid resource")
		Expect(obj.Oid).To(Equal(testoid), "DownloadCheck should report the correct oid")
		Expect(obj.Size).To(BeEquivalentTo(testcontentsz), "DownloadCheck should report the correct file size")
		Expect(obj.CanDownload()).To(BeTrue(), "Download check should report a downloadable file")
		Expect(obj.CanUpload()).To(BeFalse(), "Download check should not report an uploadable file")

		dlrdr, sz, wrerr = ctx.DownloadObject(obj)
		Expect(wrerr).To(BeNil(), "Should be no error on DownloadObject")
		Expect(sz).To(BeEquivalentTo(testcontentsz), "DownloadObject should report the correct size")
		_, err = io.CopyN(&dlbuf, dlrdr, sz)
		Expect(wrerr).To(BeNil(), "Should not be an error copying download stream")

		downloadedbytes = dlbuf.Bytes()
		Expect(downloadedbytes).To(BeEquivalentTo(testcontent), "Content downloaded from DownloadObject should be correct")

		// Now test safe fail state with DownloadCheck
		garbageoid := "99999999999999999999999999999999999"
		obj, wrerr = ctx.DownloadCheck(garbageoid)
		Expect(obj).To(BeNil(), "DownloadCheck on an invalid OID should report an nil resource")
		Expect(wrerr).ToNot(BeNil(), "DownloadCheck on an invalid OID should be an error")

		// Now batch test
		var inobjs []*lfs.ObjectResource
		inobjs = append(inobjs, &lfs.ObjectResource{Oid: testoid})
		inobjs = append(inobjs, &lfs.ObjectResource{Oid: garbageoid, Size: 500})
		retobjs, wrerr := ctx.Batch(inobjs)
		Expect(wrerr).To(BeNil(), "Should not be an error when calling Batch")
		Expect(retobjs).To(HaveLen(2), "Batch return list should be the correct length")
		for i, ro := range retobjs {
			switch i {
			case 0:
				Expect(ro.Oid).To(Equal(testoid), "OID should be correct in batch return")
				Expect(ro.Size).To(BeEquivalentTo(testcontentsz), "Size should be correct in batch return")
				Expect(ro.CanDownload()).To(BeTrue(), "First batch result should be downloadable")
				Expect(ro.CanUpload()).To(BeFalse(), "First batch result should not be uploadable")
			case 1:
				Expect(ro.Oid).To(Equal(garbageoid), "OID should be correct in batch return")
				Expect(ro.Size).To(BeEquivalentTo(500), "Size should be correct in batch return")
				Expect(ro.CanDownload()).To(BeFalse(), "First batch result should be uploadable")
				Expect(ro.CanUpload()).To(BeTrue(), "First batch result should be uploadable")
			}
		}

		ctx.Close()
	})

})
