package test

import (
	"bytes"
	"encoding/json"
	"expo-open-ota/internal/handlers"
	"expo-open-ota/internal/modules/bucket"
	"expo-open-ota/internal/services"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func cleanTest(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		projectRoot, err := findProjectRoot()
		if err != nil {
			t.Errorf("Error finding project root: %v", err)
		}
		updatesPath := filepath.Join(projectRoot, "./updates/DO_NOT_USE")
		updates, err := os.ReadDir(updatesPath)
		if err != nil {
			t.Errorf("Error reading updates directory: %v", err)
		}
		for _, update := range updates {
			if update.IsDir() {
				err = os.RemoveAll(filepath.Join(updatesPath, update.Name()))
				if err != nil {
					t.Errorf("Error removing update directory: %v", err)
				}
			}
		}
	})
}
func mockExpoResponse() {
	httpmock.RegisterResponder("POST", "https://api.expo.dev/graphql",
		func(req *http.Request) (*http.Response, error) {
			authHeader := req.Header.Get("Authorization")
			if authHeader == "Bearer expo_alternative_token" {
				return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
					"data": map[string]interface{}{
						"me": map[string]interface{}{
							"id":       "1234",
							"username": "test_alternative_username",
							"email":    "test_alternative@example.com",
						},
					},
				})
			}
			if authHeader != "Bearer expo_test_token" {
				return httpmock.NewStringResponse(http.StatusUnauthorized, `{"error": "Unauthorized"}`), nil
			}
			return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
				"data": map[string]interface{}{
					"me": map[string]interface{}{
						"id":       "123",
						"username": "test_username",
						"email":    "test@example.com",
					},
				},
			})
		})
}

func TestRequestUploadUrlWithBadEnvironment(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE?runtimeVersion=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	r.Header.Set("Authorization", "Bearer expo_test_token")
	r.URL.RawQuery = "runtimeVersion=1.0.0"
	sampleUpdatePath := filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951")
	uploadRequestsInput := ComputeUploadRequestsInput(sampleUpdatePath)
	uploadRequestsInputJSON, err := json.Marshal(uploadRequestsInput)
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 400, "Expected status code 400")
	assert.Contains(t, w.Body.String(), "Invalid environment\n", "Expected error message")
}

func TestRequestUploadUrlWithoutBearer(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "DO_NOT_USE, staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE?runtimeVersion=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	sampleUpdatePath := filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951")
	uploadRequestsInput := ComputeUploadRequestsInput(sampleUpdatePath)
	uploadRequestsInputJSON, err := json.Marshal(uploadRequestsInput)
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 401, "Expected status code 401")
	assert.Equal(t, w.Body.String(), "Invalid expo account\n", "Expected error message")
}

func TestRequestUploadUrlWithBadBearer(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "DO_NOT_USE, staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE?runtimeVersion=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	r.Header.Set("Authorization", "Bearer expo_bad_token")
	sampleUpdatePath := filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951")
	uploadRequestsInput := ComputeUploadRequestsInput(sampleUpdatePath)
	uploadRequestsInputJSON, err := json.Marshal(uploadRequestsInput)
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 401, "Expected status code 401")
	assert.Equal(t, w.Body.String(), "Invalid expo account\n", "Expected error message")
}

func TestRequestUploadUrlWithMismatchingExpoAccounts(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "DO_NOT_USE, staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE?runtimeVersion=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	r.Header.Set("Authorization", "Bearer expo_alternative_token")
	sampleUpdatePath := filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951")
	uploadRequestsInput := ComputeUploadRequestsInput(sampleUpdatePath)
	uploadRequestsInputJSON, err := json.Marshal(uploadRequestsInput)
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 401, "Expected status code 401")
	assert.Equal(t, w.Body.String(), "Invalid expo account\n", "Expected error message")
}

func TestRequestUploadUrlWithoutRuntimeVersion(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "DO_NOT_USE, staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	r.Header.Set("Authorization", "Bearer expo_test_token")
	sampleUpdatePath := filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951")
	uploadRequestsInput := ComputeUploadRequestsInput(sampleUpdatePath)
	uploadRequestsInputJSON, err := json.Marshal(uploadRequestsInput)
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 400, "Expected status code 400")
	assert.Equal(t, w.Body.String(), "No runtime version provided\n", "Expected error message")
}

func TestRequestUploadUrlWithBadRequestBody(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "DO_NOT_USE, staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE?runtimeVersion=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	r.Header.Set("Authorization", "Bearer expo_test_token")
	uploadRequestsInputJSON, err := json.Marshal(map[string]string{
		"id": "4",
	})
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 400, "Expected status code 400")
	assert.Equal(t, w.Body.String(), "No file names provided\n", "Expected error message")
}

func TestRequestUploadUrlWithBadFilenamesType(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "DO_NOT_USE, staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE?runtimeVersion=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	r.Header.Set("Authorization", "Bearer expo_test_token")
	uploadRequestsInputJSON, err := json.Marshal(map[string]int{
		"fileNames": 1,
	})
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 400, "Expected status code 400")
	assert.Equal(t, w.Body.String(), "Invalid JSON body\n", "Expected error message")
}

func TestRequestUploadUrlWithSampleUpdate(t *testing.T) {
	cleanTest(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	mockExpoResponse()
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Errorf("Error finding project root: %v", err)
	}
	os.Setenv("ENVIRONMENTS_LIST", "DO_NOT_USE, staging,production")
	os.Setenv("PUBLIC_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/public-key-test.pem"))
	os.Setenv("PRIVATE_CERT_KEY_PATH", filepath.Join(projectRoot, "/test/certs/private-key-test.pem"))
	os.Setenv("LOCAL_BUCKET_BASE_PATH", filepath.Join(projectRoot, "./updates"))
	os.Setenv("EXPO_USERNAME", "test_username")
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	q := "http://localhost:3000/requestUploadUrl/DO_NOT_USE?runtimeVersion=1"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", q, nil)
	r = mux.SetURLVars(r, map[string]string{
		"ENVIRONMENT": "DO_NOT_USE",
	})
	r.Header.Set("Authorization", "Bearer expo_test_token")
	sampleUpdatePath := filepath.Join(projectRoot, "/test/test-updates/staging/1/1674170951")
	uploadRequestsInput := ComputeUploadRequestsInput(sampleUpdatePath)
	uploadRequestsInputJSON, err := json.Marshal(uploadRequestsInput)
	if err != nil {
		t.Errorf("Error marshalling uploadRequestsInput: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(uploadRequestsInputJSON))
	handlers.RequestUploadUrlHandler(w, r)
	assert.Equal(t, w.Code, 200, "Expected status code 200")
	var fileUploadRequests []bucket.FileUploadRequest
	err = json.NewDecoder(w.Body).Decode(&fileUploadRequests)
	assert.Len(t, fileUploadRequests, 3, "Expected 3 file upload requests (1 is duplicated)")
	updateId := w.Header().Get("expo-update-id")
	assert.NotEmpty(t, updateId, "Expected non-empty update ID")
	for _, uploadRequest := range fileUploadRequests {
		requestUploadUrl := uploadRequest.RequestUploadUrl
		parsedUrl, err := url.Parse(requestUploadUrl)
		assert.Nil(t, err, "Expected valid URL")
		assert.Equal(t, parsedUrl.Scheme, "http", "Expected HTTP scheme")
		assert.Equal(t, parsedUrl.Host, "localhost:3000", "Expected localhost:3000 host")
		assert.Equal(t, parsedUrl.Path, "/uploadLocalFile", "Expected /uploadLocalFile path")
		token := parsedUrl.Query().Get("token")
		assert.NotEmpty(t, token, "Expected non-empty token")
		claims := jwt.MapClaims{}
		decoded, err := services.DecodeAndExtractJWTToken("test_jwt_secret", token, claims)
		assert.Nil(t, err, "Expected valid JWT token")
		if !decoded.Valid {
			assert.Fail(t, "Expected valid JWT token")
		}
		filePath := claims["filePath"].(string)
		assert.NotEmpty(t, filePath, "Expected non-empty file path")
		sub := claims["sub"].(string)
		assert.Equal(t, sub, "test_username", "Expected test_username sub")
	}
	// Now we can upload the files with goroutines
	var (
		ws   = make([]*httptest.ResponseRecorder, len(fileUploadRequests))
		errs = make(chan error, len(fileUploadRequests))
		wg   sync.WaitGroup
	)
	for i, uploadRequest := range fileUploadRequests {
		wg.Add(1)
		go func(index int, request bucket.FileUploadRequest) {
			defer wg.Done()
			ws[index] = httptest.NewRecorder()
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			fileBuffer, err := os.Open(projectRoot + "/test/test-updates/staging/1/1674170951/" + request.FilePath)
			if err != nil {
				fmt.Println(err)
				errs <- err
				return
			}
			part, err := writer.CreateFormFile(request.FileName, request.FileName)
			if err != nil {
				errs <- err
				return
			}
			_, err = io.Copy(part, fileBuffer)
			if err != nil {
				errs <- err
				return
			}
			err = writer.Close()
			parsedUrl, err := url.Parse(uploadRequest.RequestUploadUrl)
			token := parsedUrl.Query().Get("token")
			r := httptest.NewRequest("PUT", "/uploadLocalFile?token="+token, body)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			r.Header.Set("Authorization", "Bearer expo_test_token")
			handlers.RequestUploadLocalFileHandler(ws[index], r)
			if ws[index].Code != 200 {
				errs <- assert.AnError
			}

		}(i, uploadRequest)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.Nil(t, err, "Expected no errors")
	}
	for _, w := range ws {
		assert.Equal(t, w.Code, 200, "Expected status code 200")
		_, err := os.Open(projectRoot + "/updates/DO_NOT_USE/1/" + updateId + "/" + fileUploadRequests[0].FilePath)
		assert.Nil(t, err, "Expected no errors")
	}
}
