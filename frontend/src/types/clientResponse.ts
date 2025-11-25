import { Client } from './client';

export interface ClientResponse {
  items: {
    id: string;
    label: string;
    scopes: string[];
  }[];
  pagination: {
    total: number;
    page: number;
    per_page: number;
    total_pages: number;
  };
}
