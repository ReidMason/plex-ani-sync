package utils

import (
	"encoding/json"
)

type SliceItemNotFound struct {
}

func (s SliceItemNotFound) Error() string {
	return ""
}

func GetSliceIndex[T any](slice []T, find func(T) bool) int {
	for i, s := range slice {
		if find(s) {
			return i
		}
	}

	return -1
}

func GetSliceItem[T any](slice []T, find func(T) bool) (T, error) {
	idx := GetSliceIndex(slice, find)
	if idx != -1 {
		return slice[idx], nil
	}

	var item T
	return item, SliceItemNotFound{}
}

func ParseJson[T any](jsonData string) (item T, err error) {
	var data T
	err = json.Unmarshal([]byte(jsonData), &data)
	return data, err
}
