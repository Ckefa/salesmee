package business

import (
	"os"
	"strconv"
)

func pageSize() int {
	s := os.Getenv("TABLE_PAGE_SIZE")
	if s == "" {
		return 8
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 8
	}
	return n
}
