import { createContext, useContext, ParentComponent } from 'solid-js';
import { 
  usersState, 
  usersActions, 
  listLoading, 
  createLoading, 
  updateLoading, 
  deleteLoading 
} from '../stores/users';
import type { 
  User,
  UserListFilter,
  CreateUserRequest,
  UpdateUserRequest
} from '../types/user';

// Create the users context
interface UsersContextValue {
  // State
  state: typeof usersState;
  
  // Loading states
  loading: {
    list: () => boolean;
    create: () => boolean;
    update: () => boolean;
    delete: () => boolean;
  };
  
  // Actions
  loadUsers: (filters?: Partial<UserListFilter>) => Promise<void>;
  createUser: (userData: CreateUserRequest) => Promise<{ success: boolean; error?: string }>;
  updateUser: (userId: string, updates: UpdateUserRequest) => Promise<{ success: boolean; error?: string }>;
  deleteUser: (userId: string) => Promise<{ success: boolean; error?: string }>;
  getUser: (userId: string) => Promise<User | null>;
  selectUser: (user: User | null) => void;
  
  // Filtering and pagination
  setFilters: (filters: Partial<UserListFilter>) => void;
  setSorting: (sortBy: string, sortOrder?: 'asc' | 'desc') => void;
  setPagination: (page: number, pageSize?: number) => void;
  searchUsers: (search: string) => void;
  clearFilters: () => void;
  
  // Modal actions
  showCreateModal: () => void;
  hideCreateModal: () => void;
  showEditModal: (user: User) => void;
  hideEditModal: () => void;
  showDeleteModal: (user: User) => void;
  hideDeleteModal: () => void;
  
  // Bulk operations
  bulkDelete: (userIds: string[]) => Promise<{ success: boolean; error?: string }>;
  exportUsers: (format?: 'csv' | 'excel') => Promise<{ success: boolean; error?: string }>;
}

const UsersContext = createContext<UsersContextValue>();

// Users provider component
export const UsersProvider: ParentComponent = (props) => {
  const contextValue: UsersContextValue = {
    state: usersState,
    loading: {
      list: listLoading,
      create: createLoading,
      update: updateLoading,
      delete: deleteLoading,
    },
    loadUsers: usersActions.loadUsers,
    createUser: usersActions.createUser,
    updateUser: usersActions.updateUser,
    deleteUser: usersActions.deleteUser,
    getUser: usersActions.getUser,
    selectUser: usersActions.selectUser,
    setFilters: usersActions.setFilters,
    setSorting: usersActions.setSorting,
    setPagination: usersActions.setPagination,
    searchUsers: usersActions.searchUsers,
    clearFilters: usersActions.clearFilters,
    showCreateModal: usersActions.showCreateModal,
    hideCreateModal: usersActions.hideCreateModal,
    showEditModal: usersActions.showEditModal,
    hideEditModal: usersActions.hideEditModal,
    showDeleteModal: usersActions.showDeleteModal,
    hideDeleteModal: usersActions.hideDeleteModal,
    bulkDelete: usersActions.bulkDelete,
    exportUsers: usersActions.exportUsers,
  };

  return (
    <UsersContext.Provider value={contextValue}>
      {props.children}
    </UsersContext.Provider>
  );
};

// Hook to use users context
export const useUsers = () => {
  const context = useContext(UsersContext);
  if (!context) {
    throw new Error('useUsers must be used within a UsersProvider');
  }
  return context;
};