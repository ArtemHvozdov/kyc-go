package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentHandler(t *testing.T) {
	// Пример токена (header.payload.proof)
	token := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmcm9tIjoiVXNlciIsImJvZHkiOnsibWVzc2FnZSI6IkhlbGxvIFdvcmxkIn19.q6wIslF57Bo5Y6Czr7f7rUSC71Y-BF0fnD0Aq-01bmI`

	// Создаем запрос с телом токена
	req, err := http.NewRequest("POST", "/agent", bytes.NewBuffer([]byte(token)))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(agentHandler)

	// Выполняем запрос
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "success"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}


func TestStatusHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(statusHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "OK"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
