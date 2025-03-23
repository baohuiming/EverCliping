package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const TempDir = "./temp"

func IsExistFile(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func CreateDirectory(path string) error {
	if a, err := os.Stat(path); err != nil || !a.IsDir() {
		return os.Mkdir(path, os.ModePerm)
	}
	return nil
}

func AppendOrderToFilename(path string) string {
	dirPath := filepath.Dir(path)
	basename := filepath.Base(path) // filename with ext
	ext := filepath.Ext(basename)
	name := strings.TrimSuffix(basename, ext)

	newName := fmt.Sprintf("%s(%d)%s", name, 1, ext)

	reg := regexp.MustCompile(`^(.*)\((\d+)\)$`)
	matchResult := reg.FindSubmatch([]byte(name))

	if len(matchResult) == 3 {
		originName := string(matchResult[1])
		lastOrder, err := strconv.Atoi(string(matchResult[2]))

		if err == nil {
			newName = fmt.Sprintf("%s(%d)%s", originName, lastOrder+1, ext)
		}
	}
	return filepath.Join(dirPath, newName)
}

func LatestFilename(path string) string {
	if IsExistFile(path) {
		latest := AppendOrderToFilename(path)
		return LatestFilename(latest)
	}
	return path
}

func GetTempFilePath(filename string) string {
	if !filepath.IsAbs(TempDir) {
		// temp files path in exec path but not pwd
		tempAbsPath := path.Join(filepath.Dir(os.Args[0]), TempDir)
		return filepath.Join(tempAbsPath, filename)
	}
	return filepath.Join(TempDir, filename)
}

func SetLastFilenames(filenames []string) {
	path := GetTempFilePath("_filename.txt")
	allFilenames := strings.Join(filenames, "\n")
	_ = os.WriteFile(path, []byte(allFilenames), os.ModePerm)
}

func NewFile(path string, bytes []byte) error {
	return os.WriteFile(path, bytes, 0644)
}

func CleanTempFiles() {
	path := GetTempFilePath("_filename.txt")
	if IsExistFile(path) {
		file, err := os.Open(path)
		if err != nil {
			log.Println("failed to open temp file")
			return
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			delPath := scanner.Text()
			if err = os.Remove(delPath); err != nil {
				log.Println("failed to delete specify path")
			}
		}
	}
}
