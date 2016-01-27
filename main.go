package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello world, I'm running on %s with an %s CPU ", runtime.GOOS, runtime.GOARCH)
}

func main() {
	dat, err := ioutil.ReadFile("/etc/hosts")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(dat))

	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8080", nil)
}
