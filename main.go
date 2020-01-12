package main

import (
	"errors"
	"fmt"
	"github.com/SimonBaeumer/cmd"
	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	"github.com/thedevsaddam/gojsonq"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BorgRepositoryInfo struct {
	SizeBytes uint64
}

type BorgArchiveInfo struct {
	Name      string
	CreatedAt time.Time
}

type BorgArchiveFile struct {
	SizeBytes uint64
	Path      string
}

type BorgArchiveStats struct {
	SizeBytesByDir  map[string]uint64
	FilesCountByDir map[string]int64
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)
}

func main() {
	app := &cli.App{
		Name:      "borg-repo-stats",
		Usage:     "Print statistics about Borg Backup repository",
		ArgsUsage: "path-to-borg-repository",
		Action:    action,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func action(context *cli.Context) error {
	repositoryPath := context.Args().Get(0)
	repositoryName := filepath.Base(repositoryPath)

	repositoryInfo, err := NewRepositoryInfo(repositoryPath)
	if err != nil {
		return err
	}

	archiveInfo, err := NewArchiveInfo(repositoryPath)
	if err != nil {
		return err
	}

	archiveList, err := NewArchiveList(repositoryPath, *archiveInfo)
	if err != nil {
		return err
	}

	printSummary(repositoryName, *repositoryInfo, *archiveInfo, aggregateStats(archiveList))

	return nil
}

func NewRepositoryInfo(repositoryPath string) (*BorgRepositoryInfo, error) {
	command := cmd.NewCommand(fmt.Sprintf("borg info %q --json", repositoryPath))

	err := command.Execute()
	if err != nil {
		return nil, err
	}
	if command.ExitCode() != 0 {
		return nil, errors.New(command.Stderr())
	}
	output := command.Stdout()

	result, err := gojsonq.New().FromString(output).From("cache.stats.unique_csize").GetR()
	if err != nil {
		return nil, err
	}
	sizeBytes, err := result.Uint64()
	if err != nil {
		return nil, err
	}

	repositoryInfo := BorgRepositoryInfo{
		SizeBytes: sizeBytes,
	}

	return &repositoryInfo, nil
}

func NewArchiveInfo(repositoryPath string) (*BorgArchiveInfo, error) {
	command := cmd.NewCommand(fmt.Sprintf("borg info %q --last 1 --json", repositoryPath))

	err := command.Execute()
	if err != nil {
		return nil, err
	}
	if command.ExitCode() != 0 {
		return nil, errors.New(command.Stderr())
	}
	output := command.Stdout()

	result, err := gojsonq.New().FromString(output).From("archives.[0].name").GetR()
	if err != nil {
		return nil, err
	}
	archiveName, err := result.String()
	if err != nil {
		return nil, err
	}

	result, err = gojsonq.New().FromString(output).From("archives.[0].start").GetR()
	if err != nil {
		return nil, err
	}
	archiveCreatedAt, err := result.Time("2006-01-02T15:04:05.000000")
	if err != nil {
		return nil, err
	}

	archiveInfo := BorgArchiveInfo{
		Name:      archiveName,
		CreatedAt: archiveCreatedAt,
	}

	return &archiveInfo, nil
}

func NewArchiveList(repositoryPath string, archiveInfo BorgArchiveInfo) ([]BorgArchiveFile, error) {
	command := cmd.NewCommand(fmt.Sprintf("borg list %q::%q --json-lines", repositoryPath, archiveInfo.Name))

	err := command.Execute()
	if err != nil {
		return nil, err
	}
	if command.ExitCode() != 0 {
		return nil, errors.New(command.Stderr())
	}
	output := command.Stdout()
	outputLines := strings.Split(output, "\n")
	var files []BorgArchiveFile

	for _, line := range outputLines {
		line = strings.Trim(line, " \n\r")

		if len(line) == 0 {
			continue
		}

		result, err := gojsonq.New().FromString(line).From("size").GetR()
		if err != nil {
			return nil, err
		}
		size, err := result.Uint64()
		if err != nil {
			return nil, err
		}

		result, err = gojsonq.New().FromString(line).From("path").GetR()
		if err != nil {
			return nil, err
		}
		path, err := result.String()
		if err != nil {
			return nil, err
		}

		result, err = gojsonq.New().FromString(line).From("type").GetR()
		if err != nil {
			return nil, err
		}
		entryType, err := result.String()
		if err != nil {
			return nil, err
		}

		if entryType == "d" {
			continue
		}

		files = append(files, BorgArchiveFile{SizeBytes: size, Path: path})
	}

	return files, nil
}

func aggregateStats(filesList []BorgArchiveFile) BorgArchiveStats {
	var counts map[string]int64
	var sizeBytesByDir map[string]uint64

	counts = make(map[string]int64)
	sizeBytesByDir = make(map[string]uint64)

	for _, file := range filesList {
		tokens := strings.Split(file.Path, string(filepath.Separator))
		tokens = tokens[:len(tokens)-1] // Throw away file name

		for pathDepth := range tokens {
			path := strings.Join(tokens[:pathDepth+1], string(filepath.Separator))
			counts[path] += 1
			sizeBytesByDir[path] += file.SizeBytes
		}
	}

	return BorgArchiveStats{
		SizeBytesByDir:  sizeBytesByDir,
		FilesCountByDir: counts,
	}
}

func printSummary(
	repositoryName string,
	repositoryInfo BorgRepositoryInfo,
	info BorgArchiveInfo,
	stats BorgArchiveStats,
) {
	fmt.Printf("Repository: %s\n", repositoryName)
	fmt.Printf("Total size: %s\n", humanize.Bytes(repositoryInfo.SizeBytes))
	fmt.Printf("Archive: %s\n", info.Name)
	fmt.Printf("Created at: %s\n", humanize.Time(info.CreatedAt))
	fmt.Println()
	fmt.Println("Files by directory (the last archive only):")

	type Entry struct {
		Path      string
		Count     int64
		SizeBytes uint64
	}
	var entries []Entry
	for path, count := range stats.FilesCountByDir {
		entries = append(entries, Entry{Path: path, Count: count, SizeBytes: stats.SizeBytesByDir[path]})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return len(entries[i].Path) < len(entries[j].Path)
		}
		return entries[i].Count > entries[j].Count
	})

	for _, entry := range entries[:10] {
		fmt.Printf("%s: %d files, %s\n", entry.Path, entry.Count, humanize.Bytes(entry.SizeBytes))
	}
}
