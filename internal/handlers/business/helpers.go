package business

import (
	"salesmee/internal/config"
)

func pageSize() int {
	if config.C.TablePageSize < 1 {
		return 8
	}
	return config.C.TablePageSize
}
