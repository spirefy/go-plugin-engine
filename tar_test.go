package pluginengine

import (
	"archive/tar"
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestUntar(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pluginengine-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	sourceFile, outputPath := createTestArchive(t, tmpDir)
	err = Untar(sourceFile, outputPath)
	assertNilError(err, t)

	// Verify the extracted files
	verifyExtractedFiles(t, outputPath)
}

func TestUntar_Errors(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pluginengine-")
	if err != nil {
		t.Fatal(err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {

		}
	}(tmpDir)

	sourceFile, outputPath := createTestArchive(t, tmpDir)
	// Simulate an error during untarring
	err = os.Chmod(sourceFile, 0000)
	if err != nil {
		return
	} // Make the file unreadable

	err = Untar(sourceFile, outputPath)
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
}

func createTestArchive(t *testing.T, tmpDir string) (string, string) {
	sourceFile := filepath.Join(tmpDir, "test.tar.gz")
	outputPath := filepath.Join(tmpDir, "extracted")

	// Create a test archive
	tarFile, err := os.Create(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func(tarFile *os.File) {
		err := tarFile.Close()
		if err != nil {

		}
	}(tarFile) // Close the file

	gzipWriter := gzip.NewWriter(tarFile)

	defer func(gzipWriter *gzip.Writer) {
		err := gzipWriter.Close()
		if err != nil {

		}
	}(gzipWriter) // Close the writer

	tarWriter := tar.NewWriter(gzipWriter)

	defer func(tarWriter *tar.Writer) {
		err := tarWriter.Close()
		if err != nil {

		}
	}(tarWriter) // Close the writer

	// Add some files to the archive
	addFileToArchive(tarWriter, "file1.tar", "file1 contents")
	addFileToArchive(tarWriter, "file2.tar", "file2 contents")

	return sourceFile, outputPath
}

func verifyExtractedFiles(t *testing.T, outputPath string) {
	// Verify that the files were extracted correctly
	files, err := ioutil.ReadDir(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		fileName := filepath.Join(outputPath, file.Name())
		if file.IsDir() {
			// Verify that the directory was created correctly
			info, err := os.Stat(fileName)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode() != 0755 {
				t.Errorf("Expected directory to have mode 0755, but got %v", info.Mode())
			}
		} else {
			// Verify that the file was created correctly
			info, err := os.Stat(fileName)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode() != 0644 {
				t.Errorf("Expected file to have mode 0644, but got %v", info.Mode())
			}
		}
	}
}

func addFileToArchive(tarWriter *tar.Writer, fileName string, contents string) {
	b := []byte(contents)
	_, err := tarWriter.Write(b) // Write the file data
	if err != nil {
		return
	}
}

func assertNilError(err error, t *testing.T) {
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
}
