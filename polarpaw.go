package polarpaw

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/atotto/clipboard"
	"github.com/jessevdk/go-flags"
	"golang.org/x/tools/txtar"
)

type ExtractionStatus struct {
	ArchiveFilename string
	Success         bool
	ErrorMessage    string
}

var opts struct {
	LogFormat string `long:"log-format" choice:"text" choice:"json" default:"text" required:"false"`
	Verbose   []bool `short:"v" long:"verbose" description:"Show verbose debug information, each -v bumps log level"`
	logLevel  slog.Level
}

func Execute() int {
	if err := parseFlags(); err != nil {
		return 1
	}

	if err := setLogLevel(); err != nil {
		return 1
	}

	if err := setupLogger(); err != nil {
		return 1
	}

	if err := run(); err != nil {
		slog.Error("run failed", "error", err)
		return 1
	}

	return 0
}

func parseFlags() error {
	_, err := flags.Parse(&opts)
	return err
}

func run() error {
	var extractionStatusSlice []ExtractionStatus

	clipboardContent, err := clipboard.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading from clipboard: %w", err)
	}

	tempFile, err := os.CreateTemp("", "clipboard-*.txt")
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer func() {
		allSuccess := true
		for _, status := range extractionStatusSlice {
			if !status.Success {
				allSuccess = false
				break
			}
		}

		if allSuccess {
			if err := os.Remove(tempFile.Name()); err != nil {
				slog.Error("error deleting temp file", "error", err)
			} else {
				slog.Info("temp file deleted", "path", tempFile.Name())
			}
		}
	}()
	defer tempFile.Close()

	if _, err := tempFile.WriteString(clipboardContent); err != nil {
		return fmt.Errorf("error writing to temp file: %w", err)
	}

	archive, err := txtar.ParseFile(tempFile.Name())
	if err != nil {
		return fmt.Errorf("error parsing txtar archive: %w", err)
	}

	if len(archive.Files) < 1 {
		err := fmt.Errorf("clipboard contents not in txtar format, see %s", tempFile.Name())
		slog.Error("clipboard was not in txtar format", "error", err)
		return err
	}

	for _, file := range archive.Files {
		localPath := filepath.Join(".", file.Name)

		if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
			extractionStatusSlice = append(extractionStatusSlice, ExtractionStatus{
				ArchiveFilename: file.Name,
				Success:         false,
				ErrorMessage:    fmt.Sprintf("Error creating directories: %v", err),
			})
			continue
		}

		newFile, err := os.Create(localPath)
		if err != nil {
			extractionStatusSlice = append(extractionStatusSlice, ExtractionStatus{
				ArchiveFilename: file.Name,
				Success:         false,
				ErrorMessage:    fmt.Sprintf("Error creating new file: %v", err),
			})
			continue
		}
		defer newFile.Close()

		if _, err := newFile.Write(file.Data); err != nil {
			extractionStatusSlice = append(extractionStatusSlice, ExtractionStatus{
				ArchiveFilename: file.Name,
				Success:         false,
				ErrorMessage:    fmt.Sprintf("Error writing to file: %v", err),
			})
			continue
		}

		extractionStatusSlice = append(extractionStatusSlice, ExtractionStatus{
			ArchiveFilename: file.Name,
			Success:         true,
			ErrorMessage:    "",
		})

		slog.Debug("file extracted", "path", localPath)
	}

	return nil
}
