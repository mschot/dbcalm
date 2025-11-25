export interface Restore {
  id: number;
  start_time: string;
  end_time: string | null;
  target: 'database' | 'folder';
  target_path: string;
  backup_id: string;
  process_id: number;
}
