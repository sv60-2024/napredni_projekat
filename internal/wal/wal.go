package wal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"napredni_algoritmi_projekat/internal/blockmanager"
	"napredni_algoritmi_projekat/internal/record"
)

const (
	defaultDirectory      = "data/wal"
	defaultSegmentRecords = 10
)

type WAL struct {
	directory      string
	segmentRecords int
	blockManager   *blockmanager.BlockManager
}

func New(directory string, segmentRecords int) (*WAL, error) {
	bm, err := blockmanager.New(blockmanager.BlockSize4KB, 20)
	if err != nil {
		return nil, err
	}

	return NewWithBlockManager(directory, segmentRecords, bm)
}

func NewWithBlockManager(
	directory string,
	segmentRecords int,
	bm *blockmanager.BlockManager,
) (*WAL, error) {
	if directory == "" {
		return nil, fmt.Errorf("putanja WAL direktorijuma ne sme biti prazna")
	}

	if segmentRecords <= 0 {
		return nil, fmt.Errorf("velicina WAL segmenta mora biti veca od 0")
	}

	if bm == nil {
		return nil, fmt.Errorf("block manager ne sme biti nil")
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("greska pri kreiranju WAL direktorijuma: %w", err)
	}

	return &WAL{
		directory:      directory,
		segmentRecords: segmentRecords,
		blockManager:   bm,
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

	if len(data) > w.blockManager.BlockSize() {
		return fmt.Errorf(
			"WAL zapis ima %d bajtova i ne staje u blok od %d bajtova",
			len(data),
			w.blockManager.BlockSize(),
		)
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

		count = 0
	}

	if err := w.blockManager.WriteBlock(
		segment.path,
		count,
		data,
	); err != nil {
		return fmt.Errorf("greska pri upisu WAL zapisa: %w", err)
	}

	return nil
}

func (w *WAL) Recover() ([]record.Record, error) {
	segments, err := listSegments(w.directory)
	if err != nil {
		return nil, fmt.Errorf(
			"greska pri listanju WAL segmenata: %w",
			err,
		)
	}

	records := make([]record.Record, 0)

	for _, segment := range segments {
		segmentRecords, err := w.readSegment(segment.path)
		if err != nil {
			return nil, fmt.Errorf(
				"greska pri citanju WAL segmenta %s: %w",
				segment.path,
				err,
			)
		}

		records = append(records, segmentRecords...)
	}

	return records, nil
}

func (w *WAL) RemoveSegmentsUpTo(lastPersistedSegment int) error {
	segments, err := listSegments(w.directory)
	if err != nil {
		return fmt.Errorf(
			"greska pri listanju WAL segmenata: %w",
			err,
		)
	}

	for _, segment := range segments {
		if segment.index > lastPersistedSegment {
			continue
		}

		if err := os.Remove(segment.path); err != nil &&
			!os.IsNotExist(err) {
			return fmt.Errorf(
				"greska pri brisanju WAL segmenta %s: %w",
				segment.path,
				err,
			)
		}
	}

	return nil
}

func (w *WAL) Clear() error {
	segments, err := listSegments(w.directory)
	if err != nil {
		return fmt.Errorf(
			"greska pri listanju WAL segmenata: %w",
			err,
		)
	}

	for _, segment := range segments {
		if err := os.Remove(segment.path); err != nil &&
			!os.IsNotExist(err) {
			return fmt.Errorf(
				"greska pri brisanju WAL segmenta %s: %w",
				segment.path,
				err,
			)
		}
	}

	return nil
}

func (w *WAL) currentSegment() (segmentInfo, int, error) {
	segments, err := listSegments(w.directory)
	if err != nil {
		return segmentInfo{}, 0, fmt.Errorf(
			"greska pri listanju WAL segmenata: %w",
			err,
		)
	}

	if len(segments) == 0 {
		return segmentInfo{
			index: 0,
			path:  segmentPath(w.directory, 0),
		}, 0, nil
	}

	current := segments[len(segments)-1]

	count, err := w.countRecords(current.path)
	if err != nil {
		return segmentInfo{}, 0, fmt.Errorf(
			"greska pri brojanju zapisa u WAL segmentu: %w",
			err,
		)
	}

	return current, count, nil
}

func (w *WAL) countRecords(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}

		return 0, err
	}

	blockSize := int64(w.blockManager.BlockSize())

	if info.Size()%blockSize != 0 {
		return 0, fmt.Errorf(
			"velicina WAL segmenta nije poravnata sa velicinom bloka",
		)
	}

	return int(info.Size() / blockSize), nil
}

func (w *WAL) readSegment(path string) ([]record.Record, error) {
	recordCount, err := w.countRecords(path)
	if err != nil {
		return nil, err
	}

	records := make([]record.Record, 0, recordCount)

	for blockNumber := 0; blockNumber < recordCount; blockNumber++ {
		block, err := w.blockManager.ReadBlock(
			path,
			blockNumber,
		)
		if err != nil {
			return nil, err
		}

		rec, _, err := deserializeRecord(
			bytes.NewReader(block),
		)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return records, nil
			}

			return nil, err
		}

		records = append(records, rec)
	}

	return records, nil
}
