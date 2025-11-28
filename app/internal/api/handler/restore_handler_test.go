package handler

import (
	"net/http"
	"testing"
)

func TestListRestores(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedStatus int
		expectedCount  int   // expected number of items in response
		expectedTotal  int   // expected total in pagination
		checkFunc      func(t *testing.T, resp interface{}) // custom validation
	}{
		{
			name:           "basic listing returns all restores with default pagination",
			queryString:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
			expectedTotal:  5,
		},
		{
			name:           "filter by backup_id",
			queryString:    "?query=backup_id|backup-001",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedTotal:  1,
		},
		{
			name:           "filter by target database",
			queryString:    "?query=target|database",
			expectedStatus: http.StatusOK,
			expectedCount:  3, // 3 database restores
			expectedTotal:  3,
		},
		{
			name:           "filter by target folder",
			queryString:    "?query=target|folder",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // 2 folder restores
			expectedTotal:  2,
		},
		{
			name:           "filter by date range (Nov 5-15, 2025)",
			queryString:    "?query=start_time|gte|2025-11-05T00:00:00Z,start_time|lte|2025-11-15T23:59:59Z",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // restores on Nov 4, 9, 14
			expectedTotal:  2,
		},
		{
			name:           "order by start_time ascending",
			queryString:    "?order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
			expectedTotal:  5,
		},
		{
			name:           "order by start_time descending",
			queryString:    "?order=start_time|desc",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
			expectedTotal:  5,
		},
		{
			name:           "pagination page 1 with per_page 2",
			queryString:    "?page=1&per_page=2&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedTotal:  5,
		},
		{
			name:           "pagination page 2 with per_page 2",
			queryString:    "?page=2&per_page=2&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedTotal:  5,
		},
		{
			name:           "pagination page 3 with per_page 2 (last partial page)",
			queryString:    "?page=3&per_page=2&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // only 1 item on last page
			expectedTotal:  5,
		},
		{
			name:           "combined filters: database target in date range",
			queryString:    "?query=target|database,start_time|gte|2025-11-10T00:00:00Z&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // database restores after Nov 10
			expectedTotal:  2,
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
			queryString:    "?query=backup_id|invalidop|value",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			defer env.cleanup()
			env.seedTestData(t)

			w := env.makeRequest(t, "/restores"+tt.queryString)

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

			resp := parseRestoreListResponse(t, w)

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

func TestListRestoresPaginationMetadata(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test pagination metadata
	w := env.makeRequest(t, "/restores?page=2&per_page=2")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseRestoreListResponse(t, w)

	if resp.Pagination.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Pagination.Page)
	}
	if resp.Pagination.PerPage != 2 {
		t.Errorf("expected per_page 2, got %d", resp.Pagination.PerPage)
	}
	if resp.Pagination.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 3 { // ceil(5/2) = 3
		t.Errorf("expected total_pages 3, got %d", resp.Pagination.TotalPages)
	}
}

func TestListRestoresTargetFiltering(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test filtering by target=database
	w := env.makeRequest(t, "/restores?query=target|database&order=start_time|asc")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nBody: %s", w.Code, w.Body.String())
	}

	resp := parseRestoreListResponse(t, w)

	if len(resp.Items) != 3 {
		t.Errorf("expected 3 database restores, got %d", len(resp.Items))
	}

	// All should have target "database"
	for i, item := range resp.Items {
		if item.Target != "database" {
			t.Errorf("item[%d]: expected target 'database', got %s", i, item.Target)
		}
	}
}

func TestListRestoresBackupIDFiltering(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test filtering by specific backup_id
	w := env.makeRequest(t, "/restores?query=backup_id|backup-003")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nBody: %s", w.Code, w.Body.String())
	}

	resp := parseRestoreListResponse(t, w)

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 restore for backup-003, got %d", len(resp.Items))
		return
	}

	if resp.Items[0].BackupID != "backup-003" {
		t.Errorf("expected backup_id 'backup-003', got %s", resp.Items[0].BackupID)
	}
}
