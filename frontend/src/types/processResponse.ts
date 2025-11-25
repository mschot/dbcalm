import { Process } from "./process";
import { PaginationResponse } from "../components/Pagination";

export interface ProcessResponse {
  items: Process[];
  pagination: PaginationResponse;
}
