export interface BackupResponse {
  items: {
    id: string;
    from_backup_id?: string | null;
    start_time: string;
    end_time: string;
    schedule_id?: number | null;
    retention_value?: number | null;
    retention_unit?: string | null;
  }[];
  pagination: {
    total: number;
    page: number;
    per_page: number;
    total_pages: number;
  };
}