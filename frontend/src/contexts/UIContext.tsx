import { createContext, useContext, ParentComponent, Component } from 'solid-js';
import { uiState, uiActions } from '../stores/ui';
import type { UIState, NotificationType, ModalOptions } from '../types/ui';

// Create the UI context
interface UIContextValue {
  // State
  state: UIState;
  
  // Actions
  addNotification: (type: NotificationType, title: string, message?: string, duration?: number) => string;
  removeNotification: (id: string) => void;
  clearAllNotifications: () => void;
  
  // Convenience methods
  notifySuccess: (title: string, message?: string, duration?: number) => string;
  notifyError: (title: string, message?: string, duration?: number) => string;
  notifyWarning: (title: string, message?: string, duration?: number) => string;
  notifyInfo: (title: string, message?: string, duration?: number) => string;
  
  // Modal actions
  openModal: (component: Component<Record<string, unknown>>, props?: Record<string, unknown>, options?: ModalOptions) => string;
  closeModal: (id: string) => void;
  closeTopModal: () => void;
  closeAllModals: () => void;
  
  // Global loading
  setGlobalLoading: (loading: boolean, message?: string) => void;
}

const UIContext = createContext<UIContextValue>();

// UI provider component
export const UIProvider: ParentComponent = (props) => {
  const contextValue: UIContextValue = {
    state: uiState,
    addNotification: uiActions.addNotification,
    removeNotification: uiActions.removeNotification,
    clearAllNotifications: uiActions.clearAllNotifications,
    notifySuccess: uiActions.notifySuccess,
    notifyError: uiActions.notifyError,
    notifyWarning: uiActions.notifyWarning,
    notifyInfo: uiActions.notifyInfo,
    openModal: uiActions.openModal,
    closeModal: uiActions.closeModal,
    closeTopModal: uiActions.closeTopModal,
    closeAllModals: uiActions.closeAllModals,
    setGlobalLoading: uiActions.setGlobalLoading,
  };

  return (
    <UIContext.Provider value={contextValue}>
      {props.children}
    </UIContext.Provider>
  );
};

// Hook to use UI context
export const useUI = () => {
  const context = useContext(UIContext);
  if (!context) {
    throw new Error('useUI must be used within a UIProvider');
  }
  return context;
};