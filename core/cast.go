package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	requestCastFolder = "requests"
)

// ----------------------------------
// Pull the last event time from a live cast
// ----------------------------------
func captureCastTime(path string) (float64, bool) {
	if path == "" {
		return 0, false
	}
	line, err := tailLine(path)
	if err != nil || line == "" {
		return 0, false
	}
	var evt []interface{}
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		return 0, false
	}
	if len(evt) == 0 {
		return 0, false
	}
	val, ok := evt[0].(float64)
	return val, ok
}

// ----------------------------------
// Write a request-specific cast by slicing events
// ----------------------------------
func writeRequestCast(livePath, outPath string, start, end float64) error {
	if end < start {
		return fmt.Errorf("cast end before start")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	src, err := os.Open(livePath)
	if err != nil {
		return err
	}
	defer src.Close()

	tmpPath := outPath + ".tmp"
	dst, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = dst.Close()
	}()

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("empty cast")
	}
	if _, err := io.WriteString(dst, scanner.Text()+"\n"); err != nil {
		return err
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var evt []interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		if len(evt) == 0 {
			continue
		}
		ts, ok := evt[0].(float64)
		if !ok {
			continue
		}
		if ts < start {
			continue
		}
		if ts > end {
			break
		}
		evt[0] = ts - start
		data, err := json.Marshal(evt)
		if err != nil {
			continue
		}
		if _, err := dst.Write(data); err != nil {
			return err
		}
		if _, err := dst.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, outPath)
}

func requestCastPath(dir string, id int64) string {
	return filepath.Join(dir, fmt.Sprintf("%d.cast", id))
}

func requestCastRel(id int64) string {
	return filepath.ToSlash(filepath.Join(requestCastFolder, fmt.Sprintf("%d.cast", id)))
}

// ----------------------------------
// Read the last non-empty line from the file
// ----------------------------------
func tailLine(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() == 0 {
		return "", nil
	}

	const chunkSize int64 = 4096
	var buf []byte
	offset := info.Size()
	for offset > 0 {
		readSize := chunkSize
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize
		tmp := make([]byte, readSize)
		if _, err := file.ReadAt(tmp, offset); err != nil && err != io.EOF {
			return "", err
		}
		buf = append(tmp, buf...)
		if bytes.Contains(tmp, []byte("\n")) {
			break
		}
	}

	buf = bytes.TrimRight(buf, "\r\n")
	if len(buf) == 0 {
		return "", nil
	}
	idx := bytes.LastIndexByte(buf, '\n')
	if idx == -1 {
		return string(buf), nil
	}
	return string(buf[idx+1:]), nil
}
