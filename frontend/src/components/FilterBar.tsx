import React, { useState, useEffect } from "react";
import { format } from "date-fns";
import { DateTimePicker } from "./DateTimePicker";

interface TimeFilterConfig {
  type: 'time';
  fieldName: string;
}

interface SelectFilterConfig {
  type: 'select';
  fieldName: string;
  operator: 'eq' | 'ne' | 'in';
  placeholder?: string;
  options: Array<{value: string, label: string}>;
}

export type FilterConfig = TimeFilterConfig | SelectFilterConfig;

interface FilterBarProps {
  filters: FilterConfig[];
  onQueryChange: (queryString: string) => void;
  actionButtons?: React.ReactNode;
}

export const FilterBar: React.FC<FilterBarProps> = ({ filters, onQueryChange, actionButtons }) => {
  const [showFilters, setShowFilters] = useState<boolean>(false);
  const [filterValues, setFilterValues] = useState<Record<string, any>>({});

  useEffect(() => {
    const queryParts: string[] = [];

    filters.forEach((filter) => {
      if (filter.type === 'time') {
        const startValue = filterValues[`${filter.fieldName}_start`];
        const endValue = filterValues[`${filter.fieldName}_end`];

        if (startValue) {
          const formattedStart = format(startValue, "yyyy-MM-dd'T'HH:mm");
          queryParts.push(`${filter.fieldName}|gte|${formattedStart}`);
        }
        if (endValue) {
          const formattedEnd = format(endValue, "yyyy-MM-dd'T'HH:mm");
          queryParts.push(`${filter.fieldName}|lte|${formattedEnd}`);
        }
      } else if (filter.type === 'select') {
        const value = filterValues[filter.fieldName];
        if (value) {
          // Special handling for null checks
          if (value === 'null') {
            queryParts.push(`${filter.fieldName}|isnull`);
          } else if (value === 'not_null') {
            queryParts.push(`${filter.fieldName}|isnotnull`);
          } else {
            queryParts.push(`${filter.fieldName}|${filter.operator}|${value}`);
          }
        }
      }
    });

    onQueryChange(queryParts.join(','));
  }, [filterValues, filters, onQueryChange]);

  const handleFilterChange = (key: string, value: any) => {
    setFilterValues((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  const handleClearFilters = () => {
    setFilterValues({});
  };

  const renderTimeFilter = (filter: TimeFilterConfig) => {
    const startKey = `${filter.fieldName}_start`;
    const endKey = `${filter.fieldName}_end`;

    return (
      <React.Fragment key={filter.fieldName}>
        <DateTimePicker
          value={filterValues[startKey] || undefined}
          onChange={(date) => handleFilterChange(startKey, date)}
          className="input input-bordered w-full lg:w-48"
          placeholder="Start time"
        />
        <DateTimePicker
          value={filterValues[endKey] || undefined}
          onChange={(date) => handleFilterChange(endKey, date)}
          className="input input-bordered w-full lg:w-48"
          placeholder="End time"
        />
      </React.Fragment>
    );
  };

  const renderSelectFilter = (filter: SelectFilterConfig) => {
    return (
      <select
        key={filter.fieldName}
        value={filterValues[filter.fieldName] || ''}
        onChange={(e) => handleFilterChange(filter.fieldName, e.target.value)}
        className="select select-bordered w-full lg:w-48"
      >
        {filter.options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    );
  };

  return (
    <>
      {showFilters && (
        <div className="card bg-base-100 shadow-xl mb-4">
          <div className="card-body">
            {/* Mobile close button - top right */}
            <div className="flex justify-end mb-2 lg:hidden">
              <button
                onClick={() => setShowFilters(false)}
                className="btn btn-ghost btn-xs"
                title="Close filters"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="flex gap-4 items-center flex-wrap lg:flex-nowrap">
              {filters.map((filter) => {
                if (filter.type === 'time') {
                  return renderTimeFilter(filter);
                } else if (filter.type === 'select') {
                  return renderSelectFilter(filter);
                }
                return null;
              })}
              <button
                onClick={handleClearFilters}
                className="btn btn-ghost w-full lg:w-auto"
              >
                Clear Filters
              </button>
              {/* Desktop close button - far right inline */}
              <button
                onClick={() => setShowFilters(false)}
                className="btn btn-ghost btn-xs hidden lg:block lg:ml-auto"
                title="Close filters"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      )}

      {!showFilters && (
        <div className="flex justify-between gap-2 px-4 pt-4">
          <div className="flex gap-2">
            {actionButtons}
          </div>
          <button
            onClick={() => setShowFilters(true)}
            className="btn btn-ghost btn-sm"
            title="Toggle Filters"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
              />
            </svg>
          </button>
        </div>
      )}
    </>
  );
};
