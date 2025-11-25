
export interface Process {
  id: number;
  command: string;
  command_id: string;
  pid: number;
  status: string;
  output: string | null;
  error: string | null;
  return_code: number | null;
  start_time: string;
  end_time: string | null;
  type: string;
  args: Record<string, any>;
}
