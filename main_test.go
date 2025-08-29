package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabase is a mock implementation of DatabaseInterface
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateBuild(build *BuildRequest) (int, error) {
	args := m.Called(build)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabase) GetBuild(id int) (*BuildRequest, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BuildRequest), args.Error(1)
}

func (m *MockDatabase) ListBuilds() ([]*BuildRequest, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*BuildRequest), args.Error(1)
}

func (m *MockDatabase) UpdateBuildStatus(id int, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockDatabase) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabase) InitTables() error {
	args := m.Called()
	return args.Error(0)
}

func setupTestService() (*BuildService, *MockDatabase) {
	mockDB := new(MockDatabase)
	// Create a new registry for each test to avoid conflicts
	registry := prometheus.NewRegistry()
	service := NewBuildServiceWithRegistry(mockDB, registry)
	return service, mockDB
}

func TestHealthHandler(t *testing.T) {
	service, mockDB := setupTestService()

	tests := []struct {
		name           string
		dbPingError    error
		expectedStatus int
		expectedHealth string
	}{
		{
			name:           "healthy service",
			dbPingError:    nil,
			expectedStatus: http.StatusOK,
			expectedHealth: "healthy",
		},
		{
			name:           "unhealthy service - db error",
			dbPingError:    fmt.Errorf("database connection failed"),
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.On("Ping").Return(tt.dbPingError).Once()

			req, _ := http.NewRequest("GET", "/api/v1/health", nil)
			rr := httptest.NewRecorder()

			service.healthHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			var health map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &health)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedHealth, health["status"])
			assert.Equal(t, "build-service", health["service"])

			mockDB.AssertExpectations(t)
		})
	}
}

func TestCreateBuildHandler(t *testing.T) {
	service, mockDB := setupTestService()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		dbError        error
		expectedID     int
	}{
		{
			name: "successful build creation",
			requestBody: map[string]interface{}{
				"project_name": "test-project",
				"git_url":      "https://github.com/test/repo.git",
				"branch":       "main",
			},
			expectedStatus: http.StatusCreated,
			dbError:        nil,
			expectedID:     1,
		},
		{
			name: "missing project name",
			requestBody: map[string]interface{}{
				"git_url": "https://github.com/test/repo.git",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing git url",
			requestBody: map[string]interface{}{
				"project_name": "test-project",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "database error",
			requestBody: map[string]interface{}{
				"project_name": "test-project",
				"git_url":      "https://github.com/test/repo.git",
			},
			expectedStatus: http.StatusInternalServerError,
			dbError:        fmt.Errorf("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedStatus == http.StatusCreated || tt.dbError != nil {
				mockDB.On("CreateBuild", mock.AnythingOfType("*main.BuildRequest")).
					Return(tt.expectedID, tt.dbError).Once()
				
				// Mock the UpdateBuildStatus calls for the background processing
				if tt.dbError == nil && tt.expectedStatus == http.StatusCreated {
					mockDB.On("UpdateBuildStatus", tt.expectedID, "running").Return(nil).Maybe()
					mockDB.On("UpdateBuildStatus", tt.expectedID, mock.AnythingOfType("string")).Return(nil).Maybe()
				}
			}

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/builds", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			service.createBuildHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				var build BuildRequest
				err := json.Unmarshal(rr.Body.Bytes(), &build)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, build.ID)
				assert.Equal(t, tt.requestBody["project_name"], build.ProjectName)
				assert.Equal(t, "queued", build.Status)
				
				// Wait a moment for the goroutine to start
				time.Sleep(10 * time.Millisecond)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestGetBuildHandler(t *testing.T) {
	service, mockDB := setupTestService()

	testBuild := &BuildRequest{
		ID:          1,
		ProjectName: "test-project",
		GitURL:      "https://github.com/test/repo.git",
		Branch:      "main",
		Status:      "success",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name           string
		buildID        string
		dbResponse     *BuildRequest
		dbError        error
		expectedStatus int
	}{
		{
			name:           "successful get build",
			buildID:        "1",
			dbResponse:     testBuild,
			dbError:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "build not found",
			buildID:        "999",
			dbResponse:     nil,
			dbError:        fmt.Errorf("build not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid build ID",
			buildID:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "database error",
			buildID:        "1",
			dbResponse:     nil,
			dbError:        fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.buildID != "invalid" {
				mockDB.On("GetBuild", mock.AnythingOfType("int")).
					Return(tt.dbResponse, tt.dbError).Once()
			}

			req, _ := http.NewRequest("GET", "/api/v1/builds/"+tt.buildID, nil)
			rr := httptest.NewRecorder()

			// Setup router to handle path variables
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/builds/{id}", service.getBuildHandler).Methods("GET")
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var build BuildRequest
				err := json.Unmarshal(rr.Body.Bytes(), &build)
				assert.NoError(t, err)
				assert.Equal(t, testBuild.ID, build.ID)
				assert.Equal(t, testBuild.ProjectName, build.ProjectName)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestListBuildsHandler(t *testing.T) {
	service, mockDB := setupTestService()

	testBuilds := []*BuildRequest{
		{
			ID:          1,
			ProjectName: "project-1",
			GitURL:      "https://github.com/test/repo1.git",
			Status:      "success",
		},
		{
			ID:          2,
			ProjectName: "project-2",
			GitURL:      "https://github.com/test/repo2.git",
			Status:      "running",
		},
	}

	tests := []struct {
		name           string
		dbResponse     []*BuildRequest
		dbError        error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "successful list builds",
			dbResponse:     testBuilds,
			dbError:        nil,
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "empty list",
			dbResponse:     []*BuildRequest{},
			dbError:        nil,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "database error",
			dbResponse:     nil,
			dbError:        fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.On("ListBuilds").Return(tt.dbResponse, tt.dbError).Once()

			req, _ := http.NewRequest("GET", "/api/v1/builds", nil)
			rr := httptest.NewRecorder()

			service.listBuildsHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var builds []*BuildRequest
				err := json.Unmarshal(rr.Body.Bytes(), &builds)
				assert.NoError(t, err)
				assert.Len(t, builds, tt.expectedCount)
				
				if tt.expectedCount > 0 {
					assert.Equal(t, testBuilds[0].ProjectName, builds[0].ProjectName)
				}
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestBuildProcessing(t *testing.T) {
	service, mockDB := setupTestService()

	build := &BuildRequest{
		ID:          1,
		ProjectName: "test-project",
		Status:      "queued",
	}

	mockDB.On("UpdateBuildStatus", 1, "running").Return(nil).Once()
	mockDB.On("UpdateBuildStatus", 1, mock.MatchedBy(func(status string) bool {
		return status == "success" || status == "failed"
	})).Return(nil).Once()

	// Process build in background
	go service.processBuild(build)

	// Wait for processing to complete
	time.Sleep(3 * time.Second)

	mockDB.AssertExpectations(t)
}

func BenchmarkCreateBuild(b *testing.B) {
	service, mockDB := setupTestService()

	// Setup mock to return success for all calls
	mockDB.On("CreateBuild", mock.AnythingOfType("*main.BuildRequest")).
		Return(1, nil)

	requestBody := map[string]interface{}{
		"project_name": "benchmark-project",
		"git_url":      "https://github.com/test/repo.git",
	}

	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/builds", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		service.createBuildHandler(rr, req)
	}
}