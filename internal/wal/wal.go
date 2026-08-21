package wal

import (
	"errors"
	"fmt"
	"io"
	"os"

	"napredni_algoritmi_projekat/internal/record"
)

const (
	defaultDirectory      = "data/wal"
	defaultSegmentRecords = 10
)

type WAL struct {
	directory      string
	segmentRecords int
}

func New(directory string, segmentRecords int) (*WAL, error) {
	if directory == "" {
		return nil, fmt.Errorf("putanja WAL direktorijuma ne sme biti prazna")
	}

	if segmentRecords <= 0 {
		return nil, fmt.Errorf("velicina WAL segmenta mora biti veca od 0")
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("greska pri kreiranju WAL direktorijuma: %w", err)
	}

	return &WAL{
		directory:      directory,
		segmentRecords: segmentRecords,
	}, nil
}

func Append(rec record.Record) error {
	wal, err := New(defaultDirectory, defaultSegmentRecords)
	if err != nil {
		return err
	}

	return wal.Append(rec)
}

func Recover() ([]record.Record, error) {
	wal, err := New(defaultDirectory, defaultSegmentRecords)
	if err != nil {
		return nil, err
	}

	return wal.Recover()
}

func (w *WAL) Append(rec record.Record) error {
	data, err := serializeRecord(rec)
	if err != nil {
		return err
	}

	segment, count, err := w.currentSegment()
	if err != nil {
		return err
	}

	if count >= w.segmentRecords {
		segment = segmentInfo{
			index: segment.index + 1,
			path:  segmentPath(w.directory, segment.index+1),
		}
	}

	file, err := os.OpenFile(segment.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("greska pri otvaranju WAL segmenta: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("greska pri upisu WAL zapisa: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("greska pri sinhronizaciji WAL segmenta: %w", err)
	}

	return nil
}

func (w *WAL) Recover() ([]record.Record, error) {
	segments, err := listSegments(w.directory)
	if err != nil {
		return nil, fmt.Errorf("greska pri listanju WAL segmenata: %w", err)
	}

	records := make([]record.Record, 0)
	for _, segment := range segments {
		segmentRecords, err := readSegment(segment.path)
		if err != nil {
			return nil, fmt.Errorf("greska pri citanju WAL segmenta %s: %w", segment.path, err)
		}

		records = append(records, segmentRecords...)
	}

	return records, nil
}

func (w *WAL) RemoveSegmentsUpTo(lastPersistedSegment int) error {
	segments, err := listSegments(w.directory)
	if err != nil {
		return fmt.Errorf("greska pri listanju WAL segmenata: %w", err)
	}

	for _, segment := range segments {
		if segment.index > lastPersistedSegment {
			continue
		}

		if err := os.Remove(segment.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("greska pri brisanju WAL segmenta %s: %w", segment.path, err)
		}
	}

	return nil
}

func (w *WAL) currentSegment() (segmentInfo, int, error) {
	segments, err := listSegments(w.directory)
	if err != nil {
		return segmentInfo{}, 0, fmt.Errorf("greska pri listanju WAL segmenata: %w", err)
	}

	if len(segments) == 0 {
		return segmentInfo{
			index: 0,
			path:  segmentPath(w.directory, 0),
		}, 0, nil
	}

	current := segments[len(segments)-1]
	count, err := countRecords(current.path)
	if err != nil {
		return segmentInfo{}, 0, fmt.Errorf("greska pri brojanju zapisa u WAL segmentu: %w", err)
	}

	return current, count, nil
}

func countRecords(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	for {
		_, _, err := deserializeRecord(file)
		if err == nil {
			count++
			continue
		}

		if errors.Is(err, io.EOF) {
			return count, nil
		}

		return 0, err
	}
}

func readSegment(path string) ([]record.Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records := make([]record.Record, 0)
	for {
		rec, _, err := deserializeRecord(file)
		if err == nil {
			records = append(records, rec)
			continue
		}

		if errors.Is(err, io.EOF) {
			return records, nil
		}

		return nil, err
	}
}
