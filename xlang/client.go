package xlang

import (
	"fmt"
	"net/http"
)

func TestCaseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello from go-client")
}
