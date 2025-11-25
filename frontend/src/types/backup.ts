
export interface Backup {
  id: string;
  type: string;
  created: Date;
  retentionValue?: number | null;
  retentionUnit?: string | null;
}