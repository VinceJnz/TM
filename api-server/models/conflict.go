package models

import "errors"

var ErrStaleRecord = errors.New("record changed; reload before saving")
