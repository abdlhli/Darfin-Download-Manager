package downloader

import (
	"fmt"
	"io"
	"os"

	"darfin/internal/models"
)

// MergeSegments combines all segment temp files into the final output file
func MergeSegments(segments []models.Segment, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	for _, seg := range segments {
		if err := appendFile(outFile, seg.TempFilePath); err != nil {
			return fmt.Errorf("failed to merge segment %d: %w", seg.Index, err)
		}
	}

	return nil
}

// appendFile appends the content of srcPath to the destination writer
func appendFile(dst io.Writer, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
	return err
}

// CleanupSegments removes temporary segment files
func CleanupSegments(segments []models.Segment) {
	for _, seg := range segments {
		os.Remove(seg.TempFilePath)
	}
}
