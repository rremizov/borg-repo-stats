package main

import (
	"encoding/json"
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

type DirectoryFilesCount struct {
	Path  string
	Count int64
}

type DirectorySizeBytes struct {
	Path      string
	SizeBytes uint64
}

type Report struct {
	RepositoryName               string
	TotalSizeBytes               uint64
	LatestArchiveName            string
	LatestArchiveCreatedAt       time.Time
	LatestArchiveSizeBytesByDir  []DirectorySizeBytes
	LatestArchiveFilesCountByDir []DirectoryFilesCount
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
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output JSON",
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func action(context *cli.Context) error {
	repositoryPath := context.Args().Get(0)
	repositoryName := filepath.Base(repositoryPath)

	repositoryInfo, err := newRepositoryInfo(repositoryPath)
	if err != nil {
		return err
	}

	archiveInfo, err := newArchiveInfo(repositoryPath)
	if err != nil {
		return err
	}

	archiveList, err := newArchiveList(repositoryPath, *archiveInfo)
	if err != nil {
		return err
	}

	report := newReport(repositoryName, *repositoryInfo, *archiveInfo, aggregateStats(archiveList))

	if !context.Bool("json") {
		printTextReport(report)
		return nil
	}

	err = printJsonReport(report)
	if err != nil {
		return err
	}

	return nil
}

func newRepositoryInfo(repositoryPath string) (*BorgRepositoryInfo, error) {
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

func newArchiveInfo(repositoryPath string) (*BorgArchiveInfo, error) {
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

func newArchiveList(repositoryPath string, archiveInfo BorgArchiveInfo) ([]BorgArchiveFile, error) {
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

func newReport(repositoryName string, repositoryInfo BorgRepositoryInfo, latestArchiveInfo BorgArchiveInfo, stats BorgArchiveStats) Report {
	report := Report{
		RepositoryName:         repositoryName,
		TotalSizeBytes:         repositoryInfo.SizeBytes,
		LatestArchiveName:      latestArchiveInfo.Name,
		LatestArchiveCreatedAt: latestArchiveInfo.CreatedAt,
	}

	var sizeBytesByDir []DirectorySizeBytes
	var filesCountByDir []DirectoryFilesCount
	for path, count := range stats.FilesCountByDir {
		sizeBytesByDir = append(sizeBytesByDir, DirectorySizeBytes{Path: path, SizeBytes: stats.SizeBytesByDir[path]})
		filesCountByDir = append(filesCountByDir, DirectoryFilesCount{Path: path, Count: count})
	}

	sort.Slice(sizeBytesByDir, func(i, j int) bool {
		return sizeBytesByDir[i].SizeBytes > sizeBytesByDir[j].SizeBytes
	})
	sort.Slice(filesCountByDir, func(i, j int) bool {
		if filesCountByDir[i].Count == filesCountByDir[j].Count {
			return len(filesCountByDir[i].Path) < len(filesCountByDir[j].Path)
		}

		return filesCountByDir[i].Count > filesCountByDir[j].Count
	})

	report.LatestArchiveFilesCountByDir = filesCountByDir
	report.LatestArchiveSizeBytesByDir = sizeBytesByDir

	return report
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

func printTextReport(report Report) {
	fmt.Printf("Repository: %s\n", report.RepositoryName)
	fmt.Printf("Total size: %s\n", humanize.Bytes(report.TotalSizeBytes))
	fmt.Printf("Latest archive name: %s\n", report.LatestArchiveName)
	fmt.Printf("Latest archive created at: %s\n", humanize.Time(report.LatestArchiveCreatedAt))

	fmt.Println()
	fmt.Println("Files count by directory (the last archive only):")
	for _, entry := range report.LatestArchiveFilesCountByDir[:10] {
		fmt.Printf("%s: %d files\n", entry.Path, entry.Count)
	}

	fmt.Println()
	fmt.Println("Size by directory (the last archive only):")
	for _, entry := range report.LatestArchiveSizeBytesByDir[:10] {
		fmt.Printf("%s: %s\n", entry.Path, humanize.Bytes(entry.SizeBytes))
	}
}

func printJsonReport(report Report) error {
	b, err := json.Marshal(report)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
