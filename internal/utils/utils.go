package utils

func Contains[T comparable](slice []T, target T) bool {
	for _, element := range slice {
		if element == target {
			return true
		}
	}
	return false
}
