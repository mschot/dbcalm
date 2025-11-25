export interface RestoreResponse {
  items: {
    id: number;
    start_time: string;
    end_time: string | null;
    target: 'database' | 'folder';
    target_path: string;
    backup_id: string;
    backup_timestamp: string | null;
    process_id: number;
  }[];
  pagination: {
    total: number;
    page: number;
    per_page: number;
    total_pages: number;
  };
}
