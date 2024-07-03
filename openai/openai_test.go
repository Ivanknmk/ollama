package openai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	type testCase struct {
		Name     string
		Method   string
		Path     string
		TestPath string
		Handler  func() gin.HandlerFunc
		Endpoint func(c *gin.Context)
		Setup    func(t *testing.T, req *http.Request)
		Expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	testCases := []testCase{
		{
			Name:     "chat handler",
			Method:   http.MethodPost,
			Path:     "/api/chat",
			TestPath: "/api/chat",
			Handler:  ChatMiddleware,
			Endpoint: func(c *gin.Context) {
				var chatReq api.ChatRequest
				if err := c.ShouldBindJSON(&chatReq); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
					return
				}

				userMessage := chatReq.Messages[0].Content
				var assistantMessage string

				switch userMessage {
				case "Hello":
					assistantMessage = "Hello!"
				default:
					assistantMessage = "I'm not sure how to respond to that."
				}

				c.JSON(http.StatusOK, api.ChatResponse{
					Message: api.Message{
						Role:    "assistant",
						Content: assistantMessage,
					},
				})
			},
			Setup: func(t *testing.T, req *http.Request) {
				body := ChatCompletionRequest{
					Model:    "test-model",
					Messages: []Message{{Role: "user", Content: "Hello"}},
				}

				bodyBytes, _ := json.Marshal(body)

				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			},
			Expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)

				var chatResp ChatCompletion
				if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
					t.Fatal(err)
				}

				if chatResp.Object != "chat.completion" {
					t.Fatalf("expected chat.completion, got %s", chatResp.Object)
				}

				if chatResp.Choices[0].Message.Content != "Hello!" {
					t.Fatalf("expected Hello!, got %s", chatResp.Choices[0].Message.Content)
				}
			},
		},
		{
			Name:     "chat handler with image content",
			Method:   http.MethodPost,
			Path:     "/api/chat",
			TestPath: "/api/chat",
			Handler:  ChatMiddleware,
			Endpoint: func(c *gin.Context) {
				var chatReq api.ChatRequest
				if err := c.ShouldBindJSON(&chatReq); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
					return
				}

				userMessage := chatReq.Messages[0].Content
				userImage := chatReq.Messages[0].Images[0]
				var assistantMessage string

				img, err := base64.StdEncoding.DecodeString(imageEncoding)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
					return
				}

				switch userMessage {
				case "Hello":
					if bytes.Equal(img, userImage) {
						assistantMessage = "Hello! Nice image."
					} else {
						assistantMessage = "Hello! No image."
					}
				default:
					assistantMessage = "I'm not sure how to respond to that."
				}

				c.JSON(http.StatusOK, api.ChatResponse{
					Message: api.Message{
						Role:    "assistant",
						Content: assistantMessage,
					},
				})
			},
			Setup: func(t *testing.T, req *http.Request) {
				body := ChatCompletionRequest{
					Model: "test-model",
					Messages: []Message{
						{
							Role: "user", Content: []map[string]any{
								{"type": "text", "text": "Hello"},
								{"type": "image_url", "image_url": map[string]string{"url": imageEncoding}},
							},
						},
					},
				}

				bodyBytes, _ := json.Marshal(body)

				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			},
			Expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var chatResp ChatCompletion
				if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
					t.Fatal(err)
				}

				if chatResp.Object != "chat.completion" {
					t.Fatalf("expected chat.completion, got %s", chatResp.Object)
				}

				if chatResp.Choices[0].Message.Content != "Hello! Nice image." {
					t.Fatalf("expected Hello! Nice image., got %s", chatResp.Choices[0].Message.Content)
				}
			},
		},
		{
			Name:     "completions handler",
			Method:   http.MethodPost,
			Path:     "/api/generate",
			TestPath: "/api/generate",
			Handler:  CompletionsMiddleware,
			Endpoint: func(c *gin.Context) {
				c.JSON(http.StatusOK, api.GenerateResponse{
					Response: "Hello!",
				})
			},
			Setup: func(t *testing.T, req *http.Request) {
				body := CompletionRequest{
					Model:  "test-model",
					Prompt: "Hello",
				}

				bodyBytes, _ := json.Marshal(body)

				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			},
			Expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				var completionResp Completion
				if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
					t.Fatal(err)
				}

				if completionResp.Object != "text_completion" {
					t.Fatalf("expected text_completion, got %s", completionResp.Object)
				}

				if completionResp.Choices[0].Text != "Hello!" {
					t.Fatalf("expected Hello!, got %s", completionResp.Choices[0].Text)
				}
			},
		},
		{
			Name:     "completions handler with params",
			Method:   http.MethodPost,
			Path:     "/api/generate",
			TestPath: "/api/generate",
			Handler:  CompletionsMiddleware,
			Endpoint: func(c *gin.Context) {
				var generateReq api.GenerateRequest
				if err := c.ShouldBindJSON(&generateReq); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
					return
				}

				temperature := generateReq.Options["temperature"].(float64)
				var assistantMessage string

				switch temperature {
				case 1.6:
					assistantMessage = "Received temperature of 1.6"
				default:
					assistantMessage = fmt.Sprintf("Received temperature of %f", temperature)
				}

				c.JSON(http.StatusOK, api.GenerateResponse{
					Response: assistantMessage,
				})
			},
			Setup: func(t *testing.T, req *http.Request) {
				temp := float32(0.8)
				body := CompletionRequest{
					Model:       "test-model",
					Prompt:      "Hello",
					Temperature: &temp,
				}

				bodyBytes, _ := json.Marshal(body)

				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			},
			Expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				var completionResp Completion
				if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
					t.Fatal(err)
				}

				if completionResp.Object != "text_completion" {
					t.Fatalf("expected text_completion, got %s", completionResp.Object)
				}

				if completionResp.Choices[0].Text != "Received temperature of 1.6" {
					t.Fatalf("expected Received temperature of 1.6, got %s", completionResp.Choices[0].Text)
				}
			},
		},
		{
			Name:     "completions handler with error",
			Method:   http.MethodPost,
			Path:     "/api/generate",
			TestPath: "/api/generate",
			Handler:  CompletionsMiddleware,
			Endpoint: func(c *gin.Context) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			},
			Setup: func(t *testing.T, req *http.Request) {
				body := CompletionRequest{
					Model:  "test-model",
					Prompt: "Hello",
				}

				bodyBytes, _ := json.Marshal(body)

				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			},
			Expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				if resp.Code != http.StatusBadRequest {
					t.Fatalf("expected 400, got %d", resp.Code)
				}

				if !strings.Contains(resp.Body.String(), `"invalid request"`) {
					t.Fatalf("error was not forwarded")
				}
			},
		},
		{
			Name:     "list handler",
			Method:   http.MethodGet,
			Path:     "/api/tags",
			TestPath: "/api/tags",
			Handler:  ListMiddleware,
			Endpoint: func(c *gin.Context) {
				c.JSON(http.StatusOK, api.ListResponse{
					Models: []api.ListModelResponse{
						{
							Name: "Test Model",
						},
					},
				})
			},
			Expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)

				var listResp ListCompletion
				if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
					t.Fatal(err)
				}

				if listResp.Object != "list" {
					t.Fatalf("expected list, got %s", listResp.Object)
				}

				if len(listResp.Data) != 1 {
					t.Fatalf("expected 1, got %d", len(listResp.Data))
				}

				if listResp.Data[0].Id != "Test Model" {
					t.Fatalf("expected Test Model, got %s", listResp.Data[0].Id)
				}
			},
		},
		{
			Name:     "retrieve model",
			Method:   http.MethodGet,
			Path:     "/api/show/:model",
			TestPath: "/api/show/test-model",
			Handler:  RetrieveMiddleware,
			Endpoint: func(c *gin.Context) {
				c.JSON(http.StatusOK, api.ShowResponse{
					ModifiedAt: time.Date(2024, 6, 17, 13, 45, 0, 0, time.UTC),
				})
			},
			Expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var retrieveResp Model
				if err := json.NewDecoder(resp.Body).Decode(&retrieveResp); err != nil {
					t.Fatal(err)
				}

				if retrieveResp.Object != "model" {
					t.Fatalf("Expected object to be model, got %s", retrieveResp.Object)
				}

				if retrieveResp.Id != "test-model" {
					t.Fatalf("Expected id to be test-model, got %s", retrieveResp.Id)
				}
			},
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			router = gin.New()
			router.Use(tc.Handler())
			router.Handle(tc.Method, tc.Path, tc.Endpoint)
			req, _ := http.NewRequest(tc.Method, tc.TestPath, nil)

			if tc.Setup != nil {
				tc.Setup(t, req)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			tc.Expected(t, resp)
		})
	}
}

const imageEncoding = `iVBORw0KGgoAAAANSUhEUgAAANIAAAB4CAYAAACHHqzKAAAAAXNSR0IArs4c6QAAAIRlWElmTU0AKgAAAAgABQESAAMAAAABAAEAAAEaAAUAAAABAAAASgEb
AAUAAAABAAAAUgEoAAMAAAABAAIAAIdpAAQAAAABAAAAWgAAAAAAAABIAAAAAQAAAEgAAAABAAOgAQADAAAAAQABAACgAgAEAAAAAQAAANKgAwAEAAAAAQAA
AHgAAAAAXdsepgAAAAlwSFlzAAALEwAACxMBAJqcGAAAAVlpVFh0WE1MOmNvbS5hZG9iZS54bXAAAAAAADx4OnhtcG1ldGEgeG1sbnM6eD0iYWRvYmU6bnM6
bWV0YS8iIHg6eG1wdGs9IlhNUCBDb3JlIDYuMC4wIj4KICAgPHJkZjpSREYgeG1sbnM6cmRmPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5LzAyLzIyLXJkZi1z
eW50YXgtbnMjIj4KICAgICAgPHJkZjpEZXNjcmlwdGlvbiByZGY6YWJvdXQ9IiIKICAgICAgICAgICAgeG1sbnM6dGlmZj0iaHR0cDovL25zLmFkb2JlLmNv
bS90aWZmLzEuMC8iPgogICAgICAgICA8dGlmZjpPcmllbnRhdGlvbj4xPC90aWZmOk9yaWVudGF0aW9uPgogICAgICA8L3JkZjpEZXNjcmlwdGlvbj4KICAg
PC9yZGY6UkRGPgo8L3g6eG1wbWV0YT4KGV7hBwAAQABJREFUeAGE3QfgX9P5OP6TIRKRncgmS6aR2DNCKEKLqqpRW9FWq0q1dEQparZKF7VK7aq99yZGSCRB
BhErk0Qmyf95nZOTfOqrv/9J7ud977nnPPt5zrz3Ntp0s61XrLnmmql58+Zp6dKlqUWLFmnZsmXp888/Tx07dkwLFy5MX3zxRT4aNWqUmjVrlho3bpzatGmT
Pvnkk5y/YsWKXHfttdfOv/VauSZNmuRj0aJFSX15cIAPruS3adOmafny5Uld5dDkXP05c+akTp06pTXWWCN99tlnacmSJQGnUVp77VbpvffeS126dM4wli4t
dK8RsJoHDvUXL16cy7du3TrjXrBgQS675prNUsu1WgV/AW/ZktSxQ4dMC37BXbDgs7Q4aG7cpHFq2bJlpo984EY/3vELB94k+eqjU36V1fz580OmSyO/WZZt
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`
