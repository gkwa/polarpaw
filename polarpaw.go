package polarpaw

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/atotto/clipboard"
	"github.com/gkwa/polarpaw/version"
	"github.com/jessevdk/go-flags"
	"golang.org/x/tools/txtar"
)

// ExtractionStatus represents the extraction status of a file.
type ExtractionStatus struct {
	ArchiveFilename string
	Success         bool
	ErrorMessage    string
}

var opts struct {
	LogFormat string `long:"log-format" choice:"text" choice:"json" default:"text" required:"false"`
	Version   bool   `long:"version" required:"false"`
	Verbose   []bool `short:"v" long:"verbose" description:"Show verbose debug information, each -v bumps log level"`
	logLevel  slog.Level
}

var extractionStatusSlice []ExtractionStatus

func Execute() int {
	if err := parseFlags(); err != nil {
		return 1
	}

	if opts.Version {
		buildInfo := version.GetBuildInfo()
		fmt.Println(buildInfo)
		os.Exit(0)
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
	clipboardContent, err := clipboard.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading from clipboard: %w", err)
	}

	tempFile, err := createTempFile(clipboardContent)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer cleanupTempFile(tempFile, extractionStatusSlice)

	archive, err := parseTxtarArchive(tempFile)
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

		if err := createDirectories(localPath); err != nil {
			extractionStatusSlice = appendStatus(file.Name, false, fmt.Sprintf("Error creating directories: %v", err))
			continue
		}

		newFile, err := createFile(localPath)
		if err != nil {
			extractionStatusSlice = appendStatus(file.Name, false, fmt.Sprintf("Error creating new file: %v", err))
			continue
		}
		defer closeFile(newFile)

		if err := writeFile(newFile, file.Data); err != nil {
			extractionStatusSlice = appendStatus(file.Name, false, fmt.Sprintf("Error writing to file: %v", err))
			continue
		}

		extractionStatusSlice = appendStatus(file.Name, true, "")
		slog.Debug("file extracted", "path", localPath)
	}

	return nil
}

func createTempFile(content string) (*os.File, error) {
	tempFile, err := os.CreateTemp("", "clipboard-*.txt")
	if err != nil {
		return nil, err
	}
	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		return nil, err
	}
	return tempFile, nil
}

func cleanupTempFile(tempFile *os.File, extractionStatusSlice []ExtractionStatus) {
	allSuccess := allSuccess(extractionStatusSlice)
	if allSuccess {
		if err := os.Remove(tempFile.Name()); err != nil {
			slog.Error("error deleting temp file", "error", err)
		} else {
			slog.Info("temp file deleted", "path", tempFile.Name())
		}
	}
	tempFile.Close()
}

func parseTxtarArchive(tempFile *os.File) (*txtar.Archive, error) {
	return txtar.ParseFile(tempFile.Name())
}

func createDirectories(localPath string) error {
	return os.MkdirAll(filepath.Dir(localPath), 0o755)
}

func createFile(localPath string) (*os.File, error) {
	return os.Create(localPath)
}

func closeFile(file *os.File) {
	file.Close()
}

func writeFile(file *os.File, data []byte) error {
	_, err := file.Write(data)
	return err
}

func appendStatus(filename string, success bool, errorMessage string) []ExtractionStatus {
	return append(extractionStatusSlice, ExtractionStatus{
		ArchiveFilename: filename,
		Success:         success,
		ErrorMessage:    errorMessage,
	})
}

func allSuccess(statusSlice []ExtractionStatus) bool {
	for _, status := range statusSlice {
		if !status.Success {
			return false
		}
	}
	return true
}
