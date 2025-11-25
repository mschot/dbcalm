import React from 'react';

export interface PaginationResponse {
  total: number
  page: number
  per_page: number
  total_pages: number
}

interface PaginationProps {
  paginationResponse: PaginationResponse;
  onPageChange: (page: number) => void;
}

export const Pagination: React.FC<PaginationProps> = ({
  paginationResponse,
  onPageChange
}) => {
  const currentPage = paginationResponse.page;
  const totalPages = paginationResponse.total_pages;
  const showPagination = paginationResponse.total > paginationResponse.per_page || currentPage > 1

  return (showPagination ?
    <div className="flex justify-center py-4">
      <div className="join">
        <button
          className="join-item btn btn-sm"
          onClick={() => onPageChange(Math.max(1, currentPage - 1))}
          disabled={currentPage === 1}
        >
          «
        </button>
        <button className="join-item btn btn-sm">
          Page {currentPage} of {totalPages}
        </button>
        <button
          className="join-item btn btn-sm"
          onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
          disabled={currentPage === totalPages}
        >
          »
        </button>
      </div>
    </div>
  : null);
};
