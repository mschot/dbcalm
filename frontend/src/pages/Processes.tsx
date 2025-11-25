import React, { useState, useEffect } from "react";
import { format } from "date-fns";
import { FilterBar } from "../components/FilterBar";
import { Api } from "../utils/api";
import { Process } from "../types/process";
import { ProcessResponse } from "../types/processResponse";
import { Header } from "../components/Header";
import { Pagination, PaginationResponse } from "../components/Pagination";

const Processes = () => {
  const [processes, setProcesses] = useState<Process[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [paginationResponse, setPaginationResponse] = useState<PaginationResponse>({
    total: 0,
    page: 1,
    per_page: 25,
    total_pages: 1,
  });
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [highlightedId, setHighlightedId] = useState<number | null>(null);
  const [queryString, setQueryString] = useState<string>("");

  useEffect(() => {
    const fetchProcesses = async () => {
      try {
        let queryParams = `order=start_time|desc&page=${currentPage}`;

        if (queryString) {
          queryParams += `&query=${queryString}`;
        }

        const response = await Api.get(`/processes?${queryParams}`) as ProcessResponse;
        setProcesses(response.items);
        setPaginationResponse(response.pagination);
      } catch (error) {
        console.error("Failed to fetch processes:", error);
      }
    };

    fetchProcesses();
  }, [currentPage, queryString]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const filterId = params.get("filter");
    if (filterId && processes.length > 0) {
      const process = processes.find((p) => p.command_id === filterId);
      if (process) {
        setExpandedId(process.id);
        setHighlightedId(process.id);
        setTimeout(() => {
          const element = document.getElementById(`process-${process.id}`);
          if (element) {
            element.scrollIntoView({ behavior: "smooth", block: "center" });
          }
        }, 100);
        setTimeout(() => {
          setHighlightedId(null);
        }, 3000);
      }
    }
  }, [processes]);

  const toggleExpand = (id: number) => {
    setExpandedId(expandedId === id ? null : id);
  };

  const getStatusBadge = (status: string) => {
    if (status === "completed") {
      return <span className="badge badge-success">Success</span>;
    } else if (status === "failed") {
      return <span className="badge badge-error">Failed</span>;
    } else if (status === "running") {
      return <span className="badge badge-warning">Running</span>;
    }
    return <span className="badge">{status}</span>;
  };

  return (
    <div className="min-h-screen bg-base-200 p-4">
      <div className="container mx-auto">
        <Header currentPage="processes" />

        <div className="card bg-base-100 shadow-xl">
          <div className="card-body p-0">
            <FilterBar
              filters={[
                { type: 'time', fieldName: 'start_time' },
                {
                  type: 'select',
                  fieldName: 'type',
                  operator: 'eq',
                  options: [
                    { value: '', label: 'All types' },
                    { value: 'backup', label: 'backup' },
                    { value: 'restore', label: 'restore' }
                  ]
                }
              ]}
              onQueryChange={setQueryString}
            />
            <div>
              <table className="table table-zebra w-full">
                <thead>
                  <tr>
                    <th className="text-base-content">Type</th>
                    <th className="text-base-content">Status</th>
                    <th className="text-base-content">Start Time</th>
                    <th className="text-base-content">End Time</th>
                    <th className="text-base-content"></th>
                  </tr>
                </thead>
                <tbody>
                  {processes.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="text-center py-8 text-gray-500">
                        No processes found.
                      </td>
                    </tr>
                  ) : (
                    processes.map((process) => (
                      <React.Fragment key={process.id}>
                        <tr
                          id={`process-${process.id}`}
                          className={`hover cursor-pointer transition-colors ${
                            highlightedId === process.id ? "bg-warning bg-opacity-30" : ""
                          }`}
                          onClick={() => toggleExpand(process.id)}
                        >
                          <td className="font-medium">{process.type}</td>
                          <td>{getStatusBadge(process.status)}</td>
                          <td>{format(new Date(process.start_time), "MMM d, yyyy HH:mm:ss")}</td>
                          <td>
                            {process.end_time
                              ? format(new Date(process.end_time), "MMM d, yyyy HH:mm:ss")
                              : "-"}
                          </td>
                          <td className="text-right">
                            <svg
                              xmlns="http://www.w3.org/2000/svg"
                              className={`h-5 w-5 transition-transform ${
                                expandedId === process.id ? "rotate-180" : ""
                              }`}
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M19 9l-7 7-7-7"
                              />
                            </svg>
                          </td>
                        </tr>
                        {expandedId === process.id && (
                          <tr>
                            <td colSpan={5} className="bg-base-200">
                              <div className="p-4 space-y-4">
                                <div>
                                  <h4 className="font-semibold text-sm mb-2">Command:</h4>
                                  <pre className="bg-base-300 p-3 rounded text-sm whitespace-pre-wrap break-words overflow-wrap-anywhere">
                                    {process.command}
                                  </pre>
                                </div>
                                {process.output && (
                                  <div>
                                    <h4 className="font-semibold text-sm mb-2">Output:</h4>
                                    <pre className="bg-base-300 p-3 rounded text-sm whitespace-pre-wrap break-words overflow-wrap-anywhere max-h-64 overflow-y-auto">
                                      {process.output}
                                    </pre>
                                  </div>
                                )}
                                {process.error && (
                                  <div>
                                    <h4 className="font-semibold text-sm mb-2 text-error">
                                      Error:
                                    </h4>
                                    <pre className="bg-base-300 p-3 rounded text-sm whitespace-pre-wrap break-words overflow-wrap-anywhere max-h-64 overflow-y-auto text-error">
                                      {process.error}
                                    </pre>
                                  </div>
                                )}
                              </div>
                            </td>
                          </tr>
                        )}
                      </React.Fragment>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            <Pagination
              paginationResponse={paginationResponse}
              onPageChange={setCurrentPage}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default Processes;
