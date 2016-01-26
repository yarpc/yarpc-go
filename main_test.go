package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHandleIndexReturnsWithStatusOK(t *testing.T) {
    request, _ := http.NewRequest("GET", "/", nil)
    response := httptest.NewRecorder()

    CityHandler(response, request)

    if response.Code != http.StatusOK {
        t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
    }
}
