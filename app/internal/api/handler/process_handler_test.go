package handler

import (
	"net/http"
	"testing"
)

func TestListProcesses(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedStatus int
		expectedCount  int // expected number of items in response
		expectedTotal  int // expected total in pagination
		checkFunc      func(t *testing.T, resp interface{}) // custom validation
	}{
		{
			name:           "basic listing returns all processes with default pagination",
			queryString:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			expectedTotal:  10,
		},
		{
			name:           "filter by status success",
			queryString:    "?query=status|success",
			expectedStatus: http.StatusOK,
			expectedCount:  8, // 8 processes with status "success" (proc 1-6, 8, 10)
			expectedTotal:  8,
		},
		{
			name:           "filter by status running",
			queryString:    "?query=status|running",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // 1 process with status "running"
			expectedTotal:  1,
		},
		{
			name:           "filter by status failed",
			queryString:    "?query=status|failed",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // 1 process with status "failed"
			expectedTotal:  1,
		},
		{
			name:           "filter by type backup",
			queryString:    "?query=type|backup",
			expectedStatus: http.StatusOK,
			expectedCount:  7, // 7 backup processes
			expectedTotal:  7,
		},
		{
			name:           "filter by type restore",
			queryString:    "?query=type|restore",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // 2 restore processes
			expectedTotal:  2,
		},
		{
			name:           "filter by type cleanup_backups",
			queryString:    "?query=type|cleanup_backups",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // 1 cleanup process
			expectedTotal:  1,
		},
		{
			name:           "filter by date range (Nov 5-15, 2025)",
			queryString:    "?query=start_time|gte|2025-11-05T00:00:00Z,start_time|lte|2025-11-15T23:59:59Z",
			expectedStatus: http.StatusOK,
			expectedCount:  4, // processes on Nov 6, 9, 11, 13
			expectedTotal:  4,
		},
		{
			name:           "order by start_time ascending",
			queryString:    "?order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			expectedTotal:  10,
		},
		{
			name:           "order by start_time descending",
			queryString:    "?order=start_time|desc",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			expectedTotal:  10,
		},
		{
			name:           "order by status ascending",
			queryString:    "?order=status|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			expectedTotal:  10,
		},
		{
			name:           "pagination page 1 with per_page 3",
			queryString:    "?page=1&per_page=3&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			expectedTotal:  10,
		},
		{
			name:           "pagination page 2 with per_page 3",
			queryString:    "?page=2&per_page=3&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			expectedTotal:  10,
		},
		{
			name:           "pagination page 4 with per_page 3 (last partial page)",
			queryString:    "?page=4&per_page=3&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // only 1 item on last page
			expectedTotal:  10,
		},
		{
			name:           "combined filters: backup processes that succeeded",
			queryString:    "?query=type|backup,status|success&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  6, // 6 backup processes with status "success"
			expectedTotal:  6,
		},
		{
			name:           "combined filters: restore processes in date range",
			queryString:    "?query=type|restore,start_time|gte|2025-11-01T00:00:00Z&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // both restore processes
			expectedTotal:  2,
		},
		{
			name:           "filter by end_time isnull returns running processes",
			queryString:    "?query=end_time|isnull",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // 1 running process has no end_time
			expectedTotal:  1,
		},
		{
			name:           "filter by end_time isnotnull returns completed processes",
			queryString:    "?query=end_time|isnotnull",
			expectedStatus: http.StatusOK,
			expectedCount:  9, // 9 processes have end_time
			expectedTotal:  9,
		},
		{
			name:           "invalid query field returns 400",
			queryString:    "?query=invalid_field|value",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid order field returns 400",
			queryString:    "?order=invalid_field|desc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid operator returns 400",
			queryString:    "?query=status|invalidop|value",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			defer env.cleanup()
			env.seedTestData(t)

			w := env.makeRequest(t, "/processes"+tt.queryString)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d\nBody: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus != http.StatusOK {
				// For error cases, verify we got an error response
				errResp := parseErrorResponse(t, w)
				if errResp.Code != tt.expectedStatus {
					t.Errorf("expected error code %d, got %d", tt.expectedStatus, errResp.Code)
				}
				return
			}

			resp := parseProcessListResponse(t, w)

			if len(resp.Items) != tt.expectedCount {
				t.Errorf("expected %d items, got %d", tt.expectedCount, len(resp.Items))
			}

			if resp.Pagination.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, resp.Pagination.Total)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, resp)
			}
		})
	}
}

func TestListProcessesPaginationMetadata(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test pagination metadata
	w := env.makeRequest(t, "/processes?page=2&per_page=4")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseProcessListResponse(t, w)

	if resp.Pagination.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Pagination.Page)
	}
	if resp.Pagination.PerPage != 4 {
		t.Errorf("expected per_page 4, got %d", resp.Pagination.PerPage)
	}
	if resp.Pagination.Total != 10 {
		t.Errorf("expected total 10, got %d", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 3 { // ceil(10/4) = 3
		t.Errorf("expected total_pages 3, got %d", resp.Pagination.TotalPages)
	}
}

func TestListProcessesStatusFiltering(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test filtering by status=success
	w := env.makeRequest(t, "/processes?query=status|success&order=start_time|asc")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nBody: %s", w.Code, w.Body.String())
	}

	resp := parseProcessListResponse(t, w)

	// All should have status "success"
	for i, item := range resp.Items {
		if item.Status != "success" {
			t.Errorf("item[%d]: expected status 'success', got %s", i, item.Status)
		}
	}
}

func TestListProcessesTypeFiltering(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test filtering by type=backup
	w := env.makeRequest(t, "/processes?query=type|backup&order=start_time|asc")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nBody: %s", w.Code, w.Body.String())
	}

	resp := parseProcessListResponse(t, w)

	// All should have type "backup"
	for i, item := range resp.Items {
		if item.Type != "backup" {
			t.Errorf("item[%d]: expected type 'backup', got %s", i, item.Type)
		}
	}
}

func TestListProcessesCombinedFiltering(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test combined filtering: backup type with running status
	w := env.makeRequest(t, "/processes?query=type|backup,status|running")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nBody: %s", w.Code, w.Body.String())
	}

	resp := parseProcessListResponse(t, w)

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 running backup process, got %d", len(resp.Items))
		return
	}

	item := resp.Items[0]
	if item.Type != "backup" {
		t.Errorf("expected type 'backup', got %s", item.Type)
	}
	if item.Status != "running" {
		t.Errorf("expected status 'running', got %s", item.Status)
	}
}
