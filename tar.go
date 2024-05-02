package pluginengine

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

func Untar(sourceFile, outputPath string) error {
	// Open the compressed file
	reader, err := os.Open(sourceFile)
	if err != nil {
		return err
	}

	defer func(reader *os.File) {
		err := reader.Close()
		if err != nil {
		}
	}(reader)

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	defer func(gzipReader *gzip.Reader) {
		err := gzipReader.Close()
		if err != nil {

		}
	}(gzipReader)

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// try to create the output path in case it is not there yet
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return err
	}

	// Iterate through the files in the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		// Get the individual file name and path
		fileName := filepath.Join(outputPath, header.Name)

		// Handle directories and files differently
		switch header.Typeflag {
		case tar.TypeDir:
			// Create the directory
			if err := os.MkdirAll(fileName, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Create the file
			writer, err := os.Create(fileName)
			if err != nil {
				return err
			}

			// Copy the file data
			_, err = io.Copy(writer, tarReader)
			if err != nil {
				return err
			}

			// Close the file
			err = writer.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
