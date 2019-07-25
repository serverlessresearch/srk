package cfpackage

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Package(inputName string, outputName string, includes []string) (rerr error) {
	zf, err := os.Create(outputName)
	if err != nil {
		return err
	}
	defer func() {
		err = zf.Close()
		if err != nil && rerr == nil {
			rerr = err
		}
	}()

	zipWriter := zip.NewWriter(zf)
	defer func() {
		err = zipWriter.Close()
		if err != nil && rerr == nil {
			rerr = err
		}
	}()

	err = addPathToZip(zipWriter, inputName, "")
	if err != nil {
		return err
	}

	for _, include := range includes {
		includeFiles, exists := packagingIncludes[include]
		if !exists {
			return fmt.Errorf("unknown include %s", include)
		}
		for _, f := range includeFiles {
			err = addContentToZip(zipWriter, f)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func addContentToZip(zipWriter *zip.Writer, file includedFile) (rerr error) {
	header, err := zip.FileInfoHeader(file)
	if err != nil {
		return err
	}
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = writer.Write(file.content())
	return err
}

func addPathToZip(zipWriter *zip.Writer, fileName string, zipPath string) (rerr error) {
	var err error
	var fi os.FileInfo
	if fi, err = os.Stat(fileName); err != nil {
		return err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return fmt.Errorf("directory not implemented")
	case mode.IsRegular():
		fin, err := os.Open(fileName)
		if err != nil {
			return err
		}
		defer func() {
			err = fin.Close()
			if err != nil && rerr == nil {
				rerr = err
			}
		}()
		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}

		header.Name = filepath.Join(zipPath, filepath.Base(fileName))
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, fin)
		return err
	}
	return nil
}