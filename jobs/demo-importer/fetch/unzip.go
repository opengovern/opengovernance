package fetch

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
)

func Unzip(r io.Reader) error {
	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("error reading tar header: %v", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create a directory
			if err := os.MkdirAll(header.Name, 0755); err != nil {
				return fmt.Errorf("error creating directory: %v", err)
			}
		case tar.TypeReg:
			// Create a regular file
			outFile, err := os.Create(header.Name)
			if err != nil {
				return fmt.Errorf("error creating file: %v", err)
			}
			defer outFile.Close()

			// Copy the file content from the tar archive
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("error writing file: %v", err)
			}
		default:
			fmt.Printf("Unknown type: %c in %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}
