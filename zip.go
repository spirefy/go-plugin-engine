package pluginengine

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Unzip(sourceFile, outputPath string) error {
	// Open the zip file
	reader, err := zip.OpenReader(sourceFile)
	if err != nil {
		return err
	}

	defer func(reader *zip.ReadCloser) {
		err := reader.Close()
		if err != nil {
			fmt.Println("Error in defer close of zip")
		}
	}(reader)

	// Get the individual file name and path
	// try to create the output path in case it is not there yet
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return err
	}

	// Iterate through the files in the archive
	for _, file := range reader.File {
		// Get the individual file path
		filePath := filepath.Join(outputPath, file.Name)

		// Check for directories
		if file.FileInfo().IsDir() {
			// Create the directory
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		// Open the file within the zip
		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		defer func(fileReader io.ReadCloser) {
			err := fileReader.Close()
			if err != nil {
			}
		}(fileReader)

		// Create the target file
		targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer func(targetFile *os.File) {
			err := targetFile.Close()
			if err != nil {

			}
		}(targetFile)

		// Copy the file data
		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}

	return nil
}
