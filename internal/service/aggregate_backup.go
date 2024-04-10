package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func generateFileVersionMap(dir string) (map[string][]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fileVersionMap := make(map[string][]string)

	for _, file := range files {
		data := strings.Split(file.Name(), "_")
		table, version := strings.Join(data[:len(data)-1], "_"), data[len(data)-1]
		versions, ok := fileVersionMap[table]
		if !ok {
			fileVersionMap[table] = []string{version}
		} else {
			fileVersionMap[table] = append(versions, version)
		}
	}
	return fileVersionMap, nil
}

func parseVersion(v string) (time.Time, error) {
	format := "2006-01-02"[:len(v)-4] + ".csv"
	return time.Parse(format, v)
}

func shouldAggragate(baseVersion, version string) bool {
	tBase, errBase := parseVersion(baseVersion)
	t, err := parseVersion(version)
	if errBase != nil || err != nil {
		return false
	}
	now := time.Now().UTC().Truncate(time.Second)
	if (tBase.Year() == t.Year() && t.Year() != now.Year()) ||
		(tBase.Year() == t.Year() && tBase.Month() == t.Month() && t.Month() != now.Month()) {
		return true
	}
	return false
}

func aggregatedVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	} else if len(versions) == 1 {
		return versions[0]
	}
	baseTime, err := parseVersion(versions[0])
	if err != nil {
		return ""
	}
	sameYr, sameMonth := true, true
	for _, v := range versions {
		t, err := parseVersion(v)
		if err != nil {
			continue
		}
		sameYr = sameYr && t.Year() == baseTime.Year()
		sameMonth = sameMonth && t.Month() == baseTime.Month()
	}
	if sameYr && sameMonth {
		return baseTime.Format("2006-01.csv")
	} else if sameYr {
		return baseTime.Format("2006.csv")
	}
	return ""
}

func existContent(content []string, line string) bool {
	result := false
	for _, refLine := range content {
		result = result || refLine == line
	}
	return result
}

func aggregateContent(aggregatedContent, content []string) []string {
	result := make([]string, len(aggregatedContent))
	copy(result, aggregatedContent)
	for _, line := range content {
		if !existContent(result, line) {
			result = append(result, line)
		}
	}
	return result
}

func readBackup(dir, table, version string) []string {
	data, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%s_%s", table, version)))
	if err != nil {
		return nil
	}
	return strings.Split(string(data), "\n")
}

func saveBackup(dir, table string, versions, content []string) error {
	version := aggregatedVersion(versions)
	filename := fmt.Sprintf("%s_%s", table, version)
	data := []byte(strings.Join(content, "\n"))
	log.Debug().Str("filename", filename).Msg("save backup")
	return os.WriteFile(filepath.Join(dir, filename), data, os.ModePerm)
}

func deleteBackups(dir, table string, versions []string) error {
	for _, version := range versions {
		os.Remove(filepath.Join(dir, fmt.Sprintf("%s_%s", table, version)))
	}
	return nil
}

func aggregate(dir, table string, versions []string) error {
	sort.Strings(versions)
	baseVersion := versions[0]
	aggregatedContent := readBackup(dir, table, baseVersion)
	aggregatedVersions := []string{baseVersion}
	for _, version := range versions[1:] {
		content := readBackup(dir, table, version)
		if shouldAggragate(baseVersion, version) {
			aggregatedContent = aggregateContent(aggregatedContent, content)
			aggregatedVersions = append(aggregatedVersions, version)
		} else {
			if len(aggregatedVersions) > 1 {
				deleteBackups(dir, table, aggregatedVersions)
				saveBackup(dir, table, aggregatedVersions, aggregatedContent)
			}
			baseVersion = version
			aggregatedContent = readBackup(dir, table, baseVersion)
			aggregatedVersions = []string{baseVersion}
		}
	}
	if len(aggregatedVersions) > 1 {
		deleteBackups(dir, table, aggregatedVersions)
		saveBackup(dir, table, aggregatedVersions, aggregatedContent)
	}
	return nil
}

func AggregateBackup(dir string) error {
	fileVersionMap, err := generateFileVersionMap(dir)
	if err != nil {
		return fmt.Errorf("aggregate backup fail: %w", err)
	}

	for table, versions := range fileVersionMap {
		aggregate(dir, table, versions)
	}

	return nil
}
