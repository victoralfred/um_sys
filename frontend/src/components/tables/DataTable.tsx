import { Component, For, Show, createSignal } from 'solid-js';
import { Button } from '../buttons/Button';
import { Input } from '../ui/Input';
import { Card } from '../cards/Card';

export interface TableColumn<T = any> {
  key: keyof T;
  label: string;
  sortable?: boolean;
  render?: (value: any, row: T) => any;
  width?: string;
  align?: 'left' | 'center' | 'right';
}

export interface TableAction<T = any> {
  label: string;
  onClick: (row: T) => void;
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  icon?: string;
  disabled?: (row: T) => boolean;
}

export interface PaginationInfo {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface DataTableProps<T = any> {
  data: T[];
  columns: TableColumn<T>[];
  loading?: boolean;
  error?: string;
  actions?: TableAction<T>[];
  pagination?: PaginationInfo;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  onSort?: (column: keyof T, direction: 'asc' | 'desc') => void;
  sortColumn?: keyof T;
  sortDirection?: 'asc' | 'desc';
  selectable?: boolean;
  selectedRows?: T[];
  onSelectionChange?: (selected: T[]) => void;
  emptyMessage?: string;
  searchable?: boolean;
  onSearch?: (query: string) => void;
  searchQuery?: string;
  bulkActions?: {
    label: string;
    onClick: (selected: T[]) => void;
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  }[];
}

export const DataTable: Component<DataTableProps> = (props) => {
  const [searchQuery, setSearchQuery] = createSignal(props.searchQuery || '');

  const isAllSelected = () => {
    if (!props.selectable || !props.selectedRows) return false;
    return props.data.length > 0 && props.selectedRows.length === props.data.length;
  };


  const handleSelectAll = () => {
    if (!props.selectable || !props.onSelectionChange) return;
    
    if (isAllSelected()) {
      props.onSelectionChange([]);
    } else {
      props.onSelectionChange([...props.data]);
    }
  };

  const handleRowSelect = (row: any) => {
    if (!props.selectable || !props.onSelectionChange || !props.selectedRows) return;
    
    const isSelected = props.selectedRows.some((selected: any) => selected.id === row.id);
    
    if (isSelected) {
      props.onSelectionChange(props.selectedRows.filter((selected: any) => selected.id !== row.id));
    } else {
      props.onSelectionChange([...props.selectedRows, row]);
    }
  };

  const handleSearch = () => {
    if (props.onSearch) {
      props.onSearch(searchQuery());
    }
  };

  const handleSort = (column: TableColumn) => {
    if (!column.sortable || !props.onSort) return;
    
    let direction: 'asc' | 'desc' = 'asc';
    if (props.sortColumn === column.key && props.sortDirection === 'asc') {
      direction = 'desc';
    }
    
    props.onSort(column.key, direction);
  };

  return (
    <div>
      {/* Search and Bulk Actions */}
      <Show when={props.searchable || (props.bulkActions && props.selectedRows && props.selectedRows.length > 0)}>
        <div class="flex justify-between items-center mb-4">
          <div class="flex items-center gap-2">
            <Show when={props.searchable}>
              <div class="flex gap-2">
                <Input
                  placeholder="Search..."
                  value={searchQuery()}
                  onInput={(e) => setSearchQuery(e.currentTarget.value)}
                  onKeyPress={(e) => {
                    if (e.key === 'Enter') {
                      handleSearch();
                    }
                  }}
                  style={{ width: '300px' }}
                />
                <Button variant="secondary" onClick={handleSearch}>
                  Search
                </Button>
              </div>
            </Show>
          </div>

          <Show when={props.bulkActions && props.selectedRows && props.selectedRows.length > 0}>
            <div class="flex gap-2">
              <span class="text-body-sm flex items-center" style={{ color: "#6B778C" }}>
                {props.selectedRows!.length} selected
              </span>
              <For each={props.bulkActions}>
                {(action) => (
                  <Button
                    variant={action.variant || 'secondary'}
                    size="sm"
                    onClick={() => action.onClick(props.selectedRows!)}
                  >
                    {action.label}
                  </Button>
                )}
              </For>
            </div>
          </Show>
        </div>
      </Show>

      {/* Table */}
      <Card>
        <Show 
          when={!props.loading && !props.error}
          fallback={
            <div class="p-8 text-center">
              <Show when={props.loading}>
                <div class="spinner mb-4"></div>
                <p>Loading...</p>
              </Show>
              <Show when={props.error}>
                <p style={{ color: "#DE350B" }}>{props.error}</p>
                <Button variant="secondary" size="sm" class="mt-4">
                  Retry
                </Button>
              </Show>
            </div>
          }
        >
          <Show 
            when={props.data.length > 0}
            fallback={
              <div class="p-8 text-center">
                <p style={{ color: "#6B778C" }}>
                  {props.emptyMessage || 'No data available'}
                </p>
              </div>
            }
          >
            <div style={{ "overflow-x": "auto" }}>
              <table style={{ width: "100%", "border-collapse": "collapse" }}>
                <thead>
                  <tr style={{ "border-bottom": "2px solid #DFE1E6" }}>
                    <Show when={props.selectable}>
                      <th style={{ padding: "12px 16px", "text-align": "left", width: "40px" }}>
                        <input
                          type="checkbox"
                          checked={isAllSelected()}
                          onChange={handleSelectAll}
                        />
                      </th>
                    </Show>
                    
                    <For each={props.columns}>
                      {(column) => (
                        <th 
                          style={{ 
                            padding: "12px 16px", 
                            "text-align": column.align || "left",
                            width: column.width,
                            cursor: column.sortable ? "pointer" : "default"
                          }}
                          onClick={() => handleSort(column)}
                        >
                          <div class="flex items-center gap-2">
                            <span class="text-body-sm" style={{ "font-weight": "600" }}>
                              {column.label}
                            </span>
                            <Show when={column.sortable}>
                              <span style={{ color: "#6B778C", "font-size": "12px" }}>
                                <Show when={props.sortColumn === column.key}>
                                  {props.sortDirection === 'asc' ? '↑' : '↓'}
                                </Show>
                                <Show when={props.sortColumn !== column.key}>
                                  ↕
                                </Show>
                              </span>
                            </Show>
                          </div>
                        </th>
                      )}
                    </For>

                    <Show when={props.actions && props.actions.length > 0}>
                      <th style={{ padding: "12px 16px", "text-align": "right", width: "120px" }}>
                        <span class="text-body-sm" style={{ "font-weight": "600" }}>
                          Actions
                        </span>
                      </th>
                    </Show>
                  </tr>
                </thead>
                
                <tbody>
                  <For each={props.data}>
                    {(row) => (
                      <tr style={{ 
                        "border-bottom": "1px solid #F1F2F4",
                        "background-color": props.selectedRows?.some((selected: any) => selected.id === row.id) ? "#F4F8FF" : "transparent"
                      }}>
                        <Show when={props.selectable}>
                          <td style={{ padding: "12px 16px" }}>
                            <input
                              type="checkbox"
                              checked={props.selectedRows?.some((selected: any) => selected.id === row.id) || false}
                              onChange={() => handleRowSelect(row)}
                            />
                          </td>
                        </Show>

                        <For each={props.columns}>
                          {(column) => (
                            <td style={{ 
                              padding: "12px 16px", 
                              "text-align": column.align || "left" 
                            }}>
                              <Show 
                                when={column.render}
                                fallback={<span class="text-body">{String(row[column.key])}</span>}
                              >
                                {column.render!(row[column.key], row)}
                              </Show>
                            </td>
                          )}
                        </For>

                        <Show when={props.actions && props.actions.length > 0}>
                          <td style={{ padding: "12px 16px", "text-align": "right" }}>
                            <div class="flex gap-1 justify-end">
                              <For each={props.actions}>
                                {(action) => (
                                  <Button
                                    variant={action.variant || 'ghost'}
                                    size="sm"
                                    onClick={() => action.onClick(row)}
                                    disabled={action.disabled ? action.disabled(row) : false}
                                  >
                                    {action.label}
                                  </Button>
                                )}
                              </For>
                            </div>
                          </td>
                        </Show>
                      </tr>
                    )}
                  </For>
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            <Show when={props.pagination && props.pagination.totalPages > 1}>
              <div 
                class="flex justify-between items-center p-4"
                style={{ "border-top": "1px solid #F1F2F4" }}
              >
                <div class="text-body-sm" style={{ color: "#6B778C" }}>
                  Showing {(props.pagination!.page - 1) * props.pagination!.pageSize + 1}-{Math.min(props.pagination!.page * props.pagination!.pageSize, props.pagination!.total)} of {props.pagination!.total} items
                </div>

                <div class="flex items-center gap-4">
                  <div class="flex items-center gap-2">
                    <span class="text-body-sm">Items per page:</span>
                    <select 
                      value={props.pagination!.pageSize}
                      onChange={(e) => props.onPageSizeChange?.(parseInt(e.currentTarget.value))}
                      class="form-input"
                      style={{ width: "80px", padding: "4px 8px" }}
                    >
                      <option value="10">10</option>
                      <option value="20">20</option>
                      <option value="50">50</option>
                      <option value="100">100</option>
                    </select>
                  </div>

                  <div class="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={props.pagination!.page <= 1}
                      onClick={() => props.onPageChange?.(props.pagination!.page - 1)}
                    >
                      Previous
                    </Button>
                    
                    <span class="text-body-sm">
                      Page {props.pagination!.page} of {props.pagination!.totalPages}
                    </span>
                    
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={props.pagination!.page >= props.pagination!.totalPages}
                      onClick={() => props.onPageChange?.(props.pagination!.page + 1)}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              </div>
            </Show>
          </Show>
        </Show>
      </Card>
    </div>
  );
};