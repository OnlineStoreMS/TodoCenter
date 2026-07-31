package service

import "errors"

var (
	ErrNotFound   = errors.New("记录不存在")
	ErrBadRequest = errors.New("请求参数无效")
	ErrConflict   = errors.New("数据冲突")
)
