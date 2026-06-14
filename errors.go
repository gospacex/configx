package configx

import "errors"

var (
	ErrConfigNotFound     = errors.New("config not found")
	ErrRemoteUnavailable  = errors.New("remote config center unavailable")
	ErrLocalUnavailable   = errors.New("local config unavailable")
	ErrInvalidRemoteType  = errors.New("invalid remote type, expected apollo or consul")
	ErrKeyNotFound        = errors.New("config key not found")
)