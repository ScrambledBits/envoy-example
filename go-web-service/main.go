package main

import (
    "fmt"
    "net/http"
    "os"
)

func main() {
    hostname, err := os.Hostname()
    if err != nil {
        panic(err)
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, World! From %s", hostname)
    })

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "OK")
    })

    http.ListenAndServe(":1337", nil)
}
