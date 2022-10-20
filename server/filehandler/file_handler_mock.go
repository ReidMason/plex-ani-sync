package filehandler

import (
	"plex-ani-sync/utils"
)

type mockFileHandler[T any] struct {
	ByteData []byte
	Error    error
}

func NewMock[T any](byteData []byte, err error) *mockFileHandler[T] {
	return &mockFileHandler[T]{ByteData: byteData, Error: err}
}

func (fh mockFileHandler[T]) ReadFile(path string) ([]byte, error) {
	return fh.ByteData, fh.Error
}

func (fh mockFileHandler[T]) SaveJson(path string, data T) error {
	return nil
}

func (fh mockFileHandler[T]) LoadJsonFile(path string) (data T, err error) {
	byteValue, err := fh.ReadFile(path)

	if err != nil {
		return data, err
	}

	return utils.ParseJson[T](string(byteValue))
}

func (fh mockFileHandler[T]) CreateDirectory(path string) {
	// Implement
}

func (fh mockFileHandler[T]) CreateFile(path string) {
	// Implement
}

func (fh mockFileHandler[T]) FileExists(path string) bool {
	return false
}

func (fh mockFileHandler[T]) EnsureFileExists(path string, defaultValue T) (fileCreated bool, err error) {
	return false, nil
}
