package srk

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const inputDir = "testData"
const outputDir = "testData/testOutput"

func cleanOutput() error {
	err := os.RemoveAll(outputDir)
	if err != nil {
		return errors.Wrap(err, "Failed to clean output directory")
	}

	err = os.Mkdir(outputDir, os.ModeDir|0700)
	if err != nil {
		return errors.Wrap(err, "Failed to create output directory")
	}

	return nil
}

func checkCopiedFile(orig, new string) error {
	var err error

	var origStat os.FileInfo
	if origStat, err = os.Stat(orig); err != nil {
		return errors.Wrap(err, "Failed to stat t1")
	}

	var newStat os.FileInfo
	if newStat, err = os.Stat(new); err != nil {
		return errors.Wrap(err, "Failed to stat "+new)
	}

	// For now I think it's sufficient to make sure the files are the same size
	origSize := origStat.Size()
	newSize := newStat.Size()
	if newStat.Size() != origStat.Size() {
		return fmt.Errorf("Sizes don't match: Expected %v, Got %v\n", origSize, newSize)
	}

	return nil
}

func TestCopyFile(t *testing.T) {
	var err error

	if err = cleanOutput(); err != nil {
		t.Fatalf("%v", err)
	}

	inPath := filepath.Join(inputDir, "t1")
	outPath := filepath.Join(outputDir, "t1")
	if err = CopyFile(inPath, outPath); err != nil {
		t.Fatalf("Failed to copy file: %v\n", err)
	}

	if err := checkCopiedFile(inPath, outPath); err != nil {
		t.Fatalf("Copied file does not match original: %v\n", err)
	}
}

func TestZip(t *testing.T) {
	var err error

	if err = cleanOutput(); err != nil {
		t.Fatalf("%v", err)
	}

	err = ZipDir(inputDir, filepath.Join(inputDir, "d1"), filepath.Join(outputDir, "d1.zip"))
	if err != nil {
		t.Fatalf("Failed to zip file: %v\n", err)
	}

	fnames, err := Unzip(filepath.Join(outputDir, "d1.zip"), outputDir)
	if err != nil {
		t.Fatalf("Failed to unzip file: %v\n", err)
	}

	if len(fnames) != 1 {
		t.Fatalf("Not enough unzipped files: Expected %v, Got %v\n", 1, len(fnames))
	}

	outFilePath := filepath.Join(outputDir, "d1/t1")
	if fnames[0] != outFilePath {
		t.Fatalf("Unzipped file did not contain expected file: Expected %v, Got %v\n", outFilePath, fnames[0])
	}

	if err := checkCopiedFile(filepath.Join(inputDir, "d1", "t1"), outFilePath); err != nil {
		t.Fatalf("Unziped file does not match original: %v\n", err)
	}
}

func TestTar(t *testing.T) {
	var err error

	if err = cleanOutput(); err != nil {
		t.Fatalf("%v", err)
	}

	err = TarDir(inputDir, filepath.Join(inputDir, "d1"), filepath.Join(outputDir, "d1.tgz"))
	if err != nil {
		t.Fatalf("Failed to tar file: %v\n", err)
	}

	fnames, err := Untar(filepath.Join(outputDir, "d1.tgz"), outputDir)
	if err != nil {
		t.Fatalf("Failed to extract file: %v\n", err)
	}

	if len(fnames) != 2 {
		t.Fatalf("Not enough extracted files: Expected %v, Got %v\n", 2, len(fnames))
	}

	for i, origP := range []string{"d1", "d1/t1"} {
		outFilePath := filepath.Join(outputDir, origP)
		inFilePath := filepath.Join(inputDir, origP)
		if fnames[i] != outFilePath {
			t.Fatalf("Extracted file did not contain expected file: Expected %v, Got %v\n", outFilePath, fnames[i])
		}
		if err := checkCopiedFile(inFilePath, outFilePath); err != nil {
			t.Fatalf("Extracted file does not match original: %v\n", err)
		}
	}
}

func TestHttpPost(t *testing.T) {

	var received string
	data := "hello, world"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		received = string(data)
		w.Write(data)
	}))
	defer ts.Close()

	result, err := HttpPost(ts.URL, data)
	assert.Nil(t, err)

	assert.Equal(t, data, received)
	assert.Equal(t, data, result.String())
}

// func TestMain(m *testing.M) {
// 	var err error
//
// 	if err = cleanOutput(); err != nil {
// 		return err
// 	}
//
// 	err = os.RemoveAll(outputDir)
// 	if err != nil {
// 		fmt.Printf("Test setup failed: %v\n", err)
// 		os.Exit(1)
// 	}
//
// 	err = os.Mkdir(outputDir, os.ModeDir|0700)
// 	if err != nil {
// 		fmt.Printf("Test setup failed: %v\n", err)
// 		os.Exit(1)
// 	}
//
// 	v := m.Run()
//
// 	os.Exit(v)
// }
