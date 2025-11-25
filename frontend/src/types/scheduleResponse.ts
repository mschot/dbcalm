import { Schedule } from './schedule';

export interface ScheduleResponse {
  items: Schedule[];
  pagination: {
    total: number;
    page: number;
    per_page: number;
    total_pages: number;
  };
}
