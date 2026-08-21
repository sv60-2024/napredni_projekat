package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const segmentExtension = ".wal"

type segmentInfo struct {
	index int
	path  string
}

func segmentPath(dir string, index int) string {
	return filepath.Join(dir, fmt.Sprintf("%06d%s", index, segmentExtension))
}

func parseSegmentIndex(name string) (int, bool) {
	if filepath.Ext(name) != segmentExtension {
		return 0, false
	}

	base := strings.TrimSuffix(name, segmentExtension)
	index, err := strconv.Atoi(base)
	if err != nil {
		return 0, false
	}

	return index, true
}

func listSegments(dir string) ([]segmentInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	segments := make([]segmentInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		index, ok := parseSegmentIndex(entry.Name())
		if !ok {
			continue
		}

		segments = append(segments, segmentInfo{
			index: index,
			path:  filepath.Join(dir, entry.Name()),
		})
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].index < segments[j].index
	})

	return segments, nil
}
