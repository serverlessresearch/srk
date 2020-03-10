package srk

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
)

const inputDir = "testData"
const outputDir = "testData/testOutput"

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

func TestMain(m *testing.M) {
	err := os.RemoveAll(outputDir)
	if err != nil {
		fmt.Printf("Test setup failed: %v\n", err)
		os.Exit(1)
	}

	err = os.Mkdir(outputDir, os.ModeDir|0700)
	if err != nil {
		fmt.Printf("Test setup failed: %v\n", err)
		os.Exit(1)
	}

	v := m.Run()

	os.Exit(v)
}
