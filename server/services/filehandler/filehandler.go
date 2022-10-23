package filehandler

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"plex-ani-sync/services/utils"
)

type FileSystem[T any] interface {
	SaveJson(path string, data T) error
	LoadJsonFile(path string) (data T, err error)
	EnsureFileExists(path string, defaultValue T) (fileCreated bool, err error)
}

type FileHandler[T any] struct{}

func New[T any]() *FileHandler[T] {
	return &FileHandler[T]{}
}

func (fh FileHandler[T]) SaveJson(path string, data T) error {
	err := fh.CreateFile(path)
	if err != nil {
		return err
	}

	rawData, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, rawData, 0644)
}

func (fh FileHandler[T]) LoadJsonFile(path string) (data T, err error) {
	byteValue, err := os.ReadFile(path)

	if err != nil {
		return data, err
	}

	return utils.ParseJson[T](string(byteValue))
}

func (fh FileHandler[T]) CreateDirectory(path string) error {
	base := filepath.Dir(path)
	return os.MkdirAll(base, os.ModePerm)
}

func (fh FileHandler[T]) CreateFile(path string) error {
	err := fh.CreateDirectory(path)
	if err != nil {
		return err
	}

	_, err = os.Create(path)
	return err
}

func (fh FileHandler[T]) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func (fh FileHandler[T]) EnsureFileExists(path string, defaultValue T) (fileCreated bool, err error) {
	if !fh.FileExists(path) {
		createdError := fh.SaveJson(path, defaultValue)
		return createdError == nil, createdError
	}

	return false, nil
}
