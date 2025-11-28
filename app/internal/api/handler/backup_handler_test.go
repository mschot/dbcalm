
package handler

import (
	"net/http"
	"testing"
)

func TestListBackups(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedStatus int
		expectedCount  int      // expected number of items in response
		expectedTotal  int      // expected total in pagination
		expectedIDs    []string // expected backup IDs in order (if specified)
		checkFunc      func(t *testing.T, env *testEnv, w interface{}) // custom validation
	}{
		{
			name:           "basic listing returns all backups with default pagination",
			queryString:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			expectedTotal:  10,
		},
		{
			name:           "filter by from_backup_id isnull returns full backups only",
			queryString:    "?query=from_backup_id|isnull",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
			expectedTotal:  5,
			expectedIDs:    []string{"backup-005", "backup-004", "backup-003", "backup-002", "backup-001"}, // default order is start_time DESC
		},
		{
			name:           "filter by from_backup_id isnotnull returns incremental backups only",
			queryString:    "?query=from_backup_id|isnotnull",
			expectedStatus: http.StatusOK,
			expectedCount:  5,
			expectedTotal:  5,
			expectedIDs:    []string{"backup-010", "backup-009", "backup-008", "backup-007", "backup-006"},
		},
		{
			name:           "filter by specific id",
			queryString:    "?query=id|backup-003",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedTotal:  1,
			expectedIDs:    []string{"backup-003"},
		},
		{
			name:           "filter by date range (Nov 5-18, 2025)",
			queryString:    "?query=start_time|gte|2025-11-05T00:00:00Z,start_time|lte|2025-11-18T23:59:59Z",
			expectedStatus: http.StatusOK,
			expectedCount:  6, // backups: Nov 6 (002), 7 (007), 11 (003), 12 (008), 16 (004), 17 (009)
			expectedTotal:  6,
		},
		{
			name:           "order by start_time ascending",
			queryString:    "?order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			expectedTotal:  10,
			expectedIDs:    []string{"backup-001", "backup-006", "backup-002", "backup-007", "backup-003", "backup-008", "backup-004", "backup-009", "backup-005", "backup-010"},
		},
		{
			name:           "order by start_time descending",
			queryString:    "?order=start_time|desc",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			expectedTotal:  10,
			expectedIDs:    []string{"backup-010", "backup-005", "backup-009", "backup-004", "backup-008", "backup-003", "backup-007", "backup-002", "backup-006", "backup-001"},
		},
		{
			name:           "pagination page 1 with per_page 3",
			queryString:    "?page=1&per_page=3&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			expectedTotal:  10,
			expectedIDs:    []string{"backup-001", "backup-006", "backup-002"},
		},
		{
			name:           "pagination page 2 with per_page 3",
			queryString:    "?page=2&per_page=3&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			expectedTotal:  10,
			expectedIDs:    []string{"backup-007", "backup-003", "backup-008"},
		},
		{
			name:           "pagination page 4 with per_page 3 (last partial page)",
			queryString:    "?page=4&per_page=3&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // only 1 item on last page
			expectedTotal:  10,
			expectedIDs:    []string{"backup-010"},
		},
		{
			name:           "combined filters: full backups in date range ordered",
			queryString:    "?query=from_backup_id|isnull,start_time|gte|2025-11-10T00:00:00Z&order=start_time|asc",
			expectedStatus: http.StatusOK,
			expectedCount:  3, // backup-003, backup-004, backup-005
			expectedTotal:  3,
			expectedIDs:    []string{"backup-003", "backup-004", "backup-005"},
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
			queryString:    "?query=id|invalidop|value",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid order direction returns 400",
			queryString:    "?order=start_time|invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			defer env.cleanup()
			env.seedTestData(t)

			w := env.makeRequest(t, "/backups"+tt.queryString)

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

			resp := parseBackupListResponse(t, w)

			if len(resp.Items) != tt.expectedCount {
				t.Errorf("expected %d items, got %d", tt.expectedCount, len(resp.Items))
			}

			if resp.Pagination.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, resp.Pagination.Total)
			}

			if tt.expectedIDs != nil {
				if len(resp.Items) != len(tt.expectedIDs) {
					t.Errorf("expected %d items for ID check, got %d", len(tt.expectedIDs), len(resp.Items))
					return
				}
				for i, expectedID := range tt.expectedIDs {
					if resp.Items[i].ID != expectedID {
						t.Errorf("item[%d]: expected ID %s, got %s", i, expectedID, resp.Items[i].ID)
					}
				}
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, env, w)
			}
		})
	}
}

func TestListBackupsPaginationMetadata(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test pagination metadata
	w := env.makeRequest(t, "/backups?page=2&per_page=3")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	resp := parseBackupListResponse(t, w)

	if resp.Pagination.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Pagination.Page)
	}
	if resp.Pagination.PerPage != 3 {
		t.Errorf("expected per_page 3, got %d", resp.Pagination.PerPage)
	}
	if resp.Pagination.Total != 10 {
		t.Errorf("expected total 10, got %d", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 4 { // ceil(10/3) = 4
		t.Errorf("expected total_pages 4, got %d", resp.Pagination.TotalPages)
	}
}

func TestListBackupsDateRangeFiltering(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()
	env.seedTestData(t)

	// Test specific date range: Nov 6-16, 2025
	// backup-002: Nov 6 (full)
	// backup-007: Nov 7 (incremental from backup-002)
	// backup-003: Nov 11 (full)
	// backup-008: Nov 12 (incremental from backup-003)
	// backup-004: Nov 16 (full)
	// Note: backup-009 is Nov 17, outside range

	w := env.makeRequest(t, "/backups?query=start_time|gte|2025-11-06T00:00:00Z,start_time|lte|2025-11-16T23:59:59Z&order=start_time|asc")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d\nBody: %s", w.Code, w.Body.String())
	}

	resp := parseBackupListResponse(t, w)

	expectedIDs := []string{"backup-002", "backup-007", "backup-003", "backup-008", "backup-004"}
	if len(resp.Items) != len(expectedIDs) {
		t.Errorf("expected %d items, got %d", len(expectedIDs), len(resp.Items))
		for i, item := range resp.Items {
			t.Logf("item[%d]: %s - %v", i, item.ID, item.StartTime)
		}
		return
	}

	for i, expectedID := range expectedIDs {
		if resp.Items[i].ID != expectedID {
			t.Errorf("item[%d]: expected ID %s, got %s", i, expectedID, resp.Items[i].ID)
		}
	}
}
