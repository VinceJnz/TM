package helpers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const debugTag = "helpers."

func GetIDFromRequest(r *http.Request) (int, error) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		return 0, fmt.Errorf("invalid record ID")
	}
	return id, nil
}
