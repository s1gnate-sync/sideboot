package sysinit

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(path string) string {
	data, err := os.ReadFile(filepath.Join("/", path))
	if err != nil {
		log.Print("read", path, err)
		return ""
	}

	str := string(data)
	if str[len(str)-1] == '\n' {
		return str[0 : len(str)-1]
	}

	return str
}

func ReadPartitions() []string {
	check := false
	parts := []string{}

	for _, line := range strings.Split(ReadFile("/proc/partitions"), "\n") {
		if !check {
			check = len(line) == 0
			continue
		}

		fields := strings.Fields(line)

		partno := 0
		fmt.Sscanf(fields[1], "%d", &partno)

		blocks := 0
		fmt.Sscanf(fields[2], "%d", &blocks)

		if len(fields) == 4 && partno > 0 && blocks >= 32768 {
			parts = append(parts, fields[3])
		}
	}

	return parts
}

func ReadBytes(path string, skip uint, count uint) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	block := make([]byte, skip+count)
	if _, err := io.ReadAtLeast(f, block, len(block)); err != nil {
		return nil
	}

	return block[skip:]
}

func IsExt4(partition string) bool {
	return bytes.Equal(
		[]byte{83, 239},
		ReadBytes(filepath.Join("/dev", partition), 1080, 2),
	)
}

func FileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
