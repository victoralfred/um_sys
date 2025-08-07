import { createStore } from 'solid-js/store';
import { createUniqueId } from 'solid-js';
import type { UIState, Notification, Modal, NotificationType } from '../types/ui';

const initialState: UIState = {
  notifications: [],
  modals: [],
  isLoading: false,
  loadingMessage: undefined,
};

// Create the UI store
const [uiState, setUIState] = createStore<UIState>(initialState);

// UI actions
const uiActions = {
  // Notification actions
  addNotification: (
    type: NotificationType,
    title: string,
    message?: string,
    duration?: number,
    action?: { label: string; onClick: () => void }
  ): string => {
    const id = createUniqueId();
    const notification: Notification = {
      id,
      type,
      title,
      message,
      duration: duration !== undefined ? duration : type === 'error' ? 0 : 5000, // Errors persist by default
      action,
      createdAt: new Date(),
    };

    setUIState('notifications', notifications => [...notifications, notification]);

    // Auto-dismiss if duration is set
    if (notification.duration && notification.duration > 0) {
      if (typeof window !== 'undefined') {
        window.setTimeout(() => {
          uiActions.removeNotification(id);
        }, notification.duration);
      }
    }

    return id;
  },

  removeNotification: (id: string) => {
    setUIState('notifications', notifications => 
      notifications.filter(notification => notification.id !== id)
    );
  },

  clearAllNotifications: () => {
    setUIState('notifications', []);
  },

  // Convenience methods for different notification types
  notifySuccess: (title: string, message?: string, duration?: number) =>
    uiActions.addNotification('success', title, message, duration),

  notifyError: (title: string, message?: string, duration?: number) =>
    uiActions.addNotification('error', title, message, duration),

  notifyWarning: (title: string, message?: string, duration?: number) =>
    uiActions.addNotification('warning', title, message, duration),

  notifyInfo: (title: string, message?: string, duration?: number) =>
    uiActions.addNotification('info', title, message, duration),

  // Modal actions
  openModal: (
    component: unknown,
    props?: Record<string, unknown>,
    options?: {
      size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
      closable?: boolean;
      onClose?: () => void;
    }
  ): string => {
    const id = createUniqueId();
    const modal: Modal = {
      id,
      component,
      props: props || {},
      size: options?.size || 'md',
      closable: options?.closable !== false, // Default to true
      onClose: options?.onClose,
    };

    setUIState('modals', modals => [...modals, modal]);
    return id;
  },

  closeModal: (id: string) => {
    const modal = uiState.modals.find(m => m.id === id);
    if (modal?.onClose) {
      modal.onClose();
    }
    
    setUIState('modals', modals => 
      modals.filter(modal => modal.id !== id)
    );
  },

  closeTopModal: () => {
    const modals = uiState.modals;
    if (modals.length > 0) {
      const topModal = modals[modals.length - 1];
      uiActions.closeModal(topModal.id);
    }
  },

  closeAllModals: () => {
    // Call onClose for all modals
    uiState.modals.forEach(modal => {
      if (modal.onClose) {
        modal.onClose();
      }
    });
    
    setUIState('modals', []);
  },

  // Global loading state
  setGlobalLoading: (loading: boolean, message?: string) => {
    setUIState('isLoading', loading);
    setUIState('loadingMessage', message);
  },

  // Keyboard shortcuts
  handleKeyPress: (event: { key: string }) => {
    // ESC key to close top modal
    if (event.key === 'Escape' && uiState.modals.length > 0) {
      const topModal = uiState.modals[uiState.modals.length - 1];
      if (topModal.closable) {
        uiActions.closeModal(topModal.id);
      }
    }
  },
};

// Convenience hooks for common UI patterns
const createConfirmModal = (
  title: string,
  message: string,
  onConfirm: () => void | Promise<void>,
  options?: {
    confirmText?: string;
    cancelText?: string;
    type?: 'danger' | 'warning' | 'info';
  }
) => {
  return uiActions.openModal(
    // TODO: Create ConfirmModal component
    'ConfirmModal',
    {
      title,
      message,
      onConfirm,
      confirmText: options?.confirmText || 'Confirm',
      cancelText: options?.cancelText || 'Cancel',
      type: options?.type || 'info',
    },
    { size: 'sm' }
  );
};

const createFormModal = (
  component: unknown,
  props: Record<string, unknown>,
  options?: {
    size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
    title?: string;
  }
) => {
  return uiActions.openModal(
    component,
    props,
    { size: options?.size || 'md' }
  );
};

// Auto-cleanup: Remove old notifications (older than 1 hour)
if (typeof window !== 'undefined') {
  window.setInterval(() => {
    const oneHourAgo = new Date(Date.now() - 60 * 60 * 1000);
    setUIState('notifications', notifications =>
      notifications.filter(notification => notification.createdAt > oneHourAgo)
    );
  }, 60000); // Check every minute
}

export {
  uiState,
  uiActions,
  createConfirmModal,
  createFormModal,
};