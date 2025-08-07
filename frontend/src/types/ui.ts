// UI state types for modals, notifications, etc.

export type NotificationType = 'success' | 'error' | 'warning' | 'info';

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message?: string;
  duration?: number; // in milliseconds, null for persistent
  action?: {
    label: string;
    onClick: () => void;
  };
  createdAt: Date;
}

import type { Component, JSX } from 'solid-js';

export interface ModalOptions {
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
  closable?: boolean;
  onClose?: () => void;
}

export interface Modal {
  id: string;
  component: Component<Record<string, unknown>>; // SolidJS component
  props?: Record<string, unknown>;
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
  closable?: boolean;
  onClose?: () => void;
}

export interface UIState {
  notifications: Notification[];
  modals: Modal[];
  isLoading: boolean;
  loadingMessage?: string;
}

export interface PaginationState {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface TableState<T = unknown> {
  data: T[];
  loading: boolean;
  error?: string;
  pagination: PaginationState;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  filters?: Record<string, unknown>;
}