export interface Schedule {
  id: number;
  backup_type: "full" | "incremental";
  frequency: "daily" | "weekly" | "monthly" | "hourly" | "interval";
  day_of_week?: number;  // 0-6 (0=Sunday)
  day_of_month?: number; // 1-28
  hour?: number;         // 0-23, only for non-interval schedules
  minute?: number;       // 0-59, for non-interval schedules and hourly
  interval_value?: number; // interval value (e.g., 15, 30, 2)
  interval_unit?: "minutes" | "hours"; // interval unit
  retention_value?: number; // retention value (e.g., 7, 30, 52)
  retention_unit?: "days" | "weeks" | "months"; // retention unit
  enabled: boolean;
  created_at: string;
  updated_at: string;
}
