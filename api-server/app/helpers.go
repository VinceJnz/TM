package app

import "log"

type DebugWriter struct {
}

func (d DebugWriter) Write(p []byte) (int, error) {
	log.Printf("debugWrite()1 p: %+v", string(p))
	return 0, nil
}
