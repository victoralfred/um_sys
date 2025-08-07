import { createStore } from 'solid-js/store';
import { createSignal } from 'solid-js';
import { usersService } from '../services/users';
import type {
  User,
  UserListFilter,
  CreateUserRequest,
  UpdateUserRequest,
  PaginationState,
} from '../types';

interface UsersState {
  // User list state
  users: User[];
  selectedUser: User | null;
  loading: boolean;
  error: string | null;
  
  // Pagination
  pagination: PaginationState;
  
  // Filtering and sorting
  filters: UserListFilter;
  sortBy: string;
  sortOrder: 'asc' | 'desc';
  
  // UI state
  showCreateModal: boolean;
  showEditModal: boolean;
  showDeleteModal: boolean;
}

const initialState: UsersState = {
  users: [],
  selectedUser: null,
  loading: false,
  error: null,
  pagination: {
    page: 1,
    pageSize: 20,
    total: 0,
    totalPages: 0,
  },
  filters: {
    search: '',
    sortBy: 'createdAt',
    sortOrder: 'desc',
  },
  sortBy: 'createdAt',
  sortOrder: 'desc',
  showCreateModal: false,
  showEditModal: false,
  showDeleteModal: false,
};

// Create the users store
const [usersState, setUsersState] = createStore<UsersState>(initialState);

// Create signals for reactive loading states
const [listLoading, setListLoading] = createSignal(false);
const [createLoading, setCreateLoading] = createSignal(false);
const [updateLoading, setUpdateLoading] = createSignal(false);
const [deleteLoading, setDeleteLoading] = createSignal(false);

// Users actions
const usersActions = {
  // Load users list
  loadUsers: async (filters?: Partial<UserListFilter>): Promise<void> => {
    setListLoading(true);
    setUsersState('loading', true);
    setUsersState('error', null);

    try {
      // Merge filters with current state
      const currentFilters = { ...usersState.filters, ...filters };
      setUsersState('filters', currentFilters);

      const data = await usersService.getUsers(currentFilters);

      if (data.success && data.data) {
        setUsersState('users', data.data.users);
        setUsersState('pagination', {
          page: data.data.page,
          pageSize: data.data.pageSize,
          total: data.data.total,
          totalPages: data.data.totalPages,
        });
      } else {
        setUsersState('error', data.error?.message || 'Failed to load users');
      }
    } catch (error) {
      console.error('Error loading users:', error);
      setUsersState('error', 'Failed to connect to server');
    } finally {
      setListLoading(false);
      setUsersState('loading', false);
    }
  },

  // Create a new user
  createUser: async (userData: CreateUserRequest): Promise<{ success: boolean; error?: string }> => {
    setCreateLoading(true);
    
    try {
      const data = await usersService.createUser(userData);

      if (data.success) {
        // Reload the users list
        await usersActions.loadUsers();
        setUsersState('showCreateModal', false);
        return { success: true };
      } else {
        return { success: false, error: data.error?.message || 'Failed to create user' };
      }
    } catch (error) {
      console.error('Error creating user:', error);
      return { success: false, error: error instanceof Error ? error.message : 'Failed to connect to server' };
    } finally {
      setCreateLoading(false);
    }
  },

  // Update a user
  updateUser: async (userId: string, updates: UpdateUserRequest): Promise<{ success: boolean; error?: string }> => {
    setUpdateLoading(true);
    
    try {
      const data = await usersService.updateUser(userId, updates);

      if (data.success) {
        // Update the user in the list
        const updatedUsers = usersState.users.map(user => 
          user.id === userId ? { ...user, ...updates } : user
        );
        setUsersState('users', updatedUsers);
        
        // Update selected user if it's the one being updated
        if (usersState.selectedUser?.id === userId) {
          setUsersState('selectedUser', { ...usersState.selectedUser, ...updates });
        }
        
        setUsersState('showEditModal', false);
        return { success: true };
      } else {
        return { success: false, error: data.error?.message || 'Failed to update user' };
      }
    } catch (error) {
      console.error('Error updating user:', error);
      return { success: false, error: error instanceof Error ? error.message : 'Failed to connect to server' };
    } finally {
      setUpdateLoading(false);
    }
  },

  // Delete a user
  deleteUser: async (userId: string): Promise<{ success: boolean; error?: string }> => {
    setDeleteLoading(true);
    
    try {
      const data = await usersService.deleteUser(userId);

      if (data.success) {
        // Remove the user from the list
        const updatedUsers = usersState.users.filter(user => user.id !== userId);
        setUsersState('users', updatedUsers);
        
        // Clear selected user if it's the one being deleted
        if (usersState.selectedUser?.id === userId) {
          setUsersState('selectedUser', null);
        }
        
        setUsersState('showDeleteModal', false);
        return { success: true };
      } else {
        return { success: false, error: data.error?.message || 'Failed to delete user' };
      }
    } catch (error) {
      console.error('Error deleting user:', error);
      return { success: false, error: error instanceof Error ? error.message : 'Failed to connect to server' };
    } finally {
      setDeleteLoading(false);
    }
  },

  // Get a single user
  getUser: async (userId: string): Promise<User | null> => {
    try {
      const data = await usersService.getUser(userId);
      return data.success && data.data ? data.data : null;
    } catch (error) {
      console.error('Error fetching user:', error);
      return null;
    }
  },

  // Select a user
  selectUser: (user: User | null) => {
    setUsersState('selectedUser', user);
  },

  // Set filters
  setFilters: (filters: Partial<UserListFilter>) => {
    setUsersState('filters', { ...usersState.filters, ...filters });
    // Reset pagination when filtering
    setUsersState('pagination', 'page', 1);
    // Reload users with new filters
    usersActions.loadUsers(filters);
  },

  // Set sorting
  setSorting: (sortBy: string, sortOrder: 'asc' | 'desc' = 'asc') => {
    setUsersState('sortBy', sortBy);
    setUsersState('sortOrder', sortOrder);
    setUsersState('filters', 'sortBy', sortBy);
    setUsersState('filters', 'sortOrder', sortOrder);
    // Reload users with new sorting
    usersActions.loadUsers();
  },

  // Set pagination
  setPagination: (page: number, pageSize?: number) => {
    const newPageSize = pageSize || usersState.pagination.pageSize;
    const offset = (page - 1) * newPageSize;
    
    setUsersState('pagination', 'page', page);
    if (pageSize) {
      setUsersState('pagination', 'pageSize', pageSize);
    }
    
    // Update filters with new pagination
    setUsersState('filters', 'offset', offset);
    setUsersState('filters', 'limit', newPageSize);
    
    // Reload users
    usersActions.loadUsers();
  },

  // Search users
  searchUsers: (search: string) => {
    setUsersState('filters', 'search', search);
    setUsersState('pagination', 'page', 1); // Reset to first page
    usersActions.loadUsers();
  },

  // Clear filters
  clearFilters: () => {
    setUsersState('filters', initialState.filters);
    setUsersState('pagination', 'page', 1);
    usersActions.loadUsers();
  },

  // Modal actions
  showCreateModal: () => setUsersState('showCreateModal', true),
  hideCreateModal: () => setUsersState('showCreateModal', false),
  
  showEditModal: (user: User) => {
    setUsersState('selectedUser', user);
    setUsersState('showEditModal', true);
  },
  hideEditModal: () => setUsersState('showEditModal', false),
  
  showDeleteModal: (user: User) => {
    setUsersState('selectedUser', user);
    setUsersState('showDeleteModal', true);
  },
  hideDeleteModal: () => setUsersState('showDeleteModal', false),

  // Bulk actions
  bulkDelete: async (userIds: string[]): Promise<{ success: boolean; error?: string }> => {
    try {
      const data = await usersService.bulkDeleteUsers(userIds);
      
      if (data.success) {
        // Remove deleted users from the list
        const updatedUsers = usersState.users.filter(user => !userIds.includes(user.id));
        setUsersState('users', updatedUsers);
        return { success: true };
      } else {
        return { success: false, error: data.error?.message || 'Failed to delete users' };
      }
    } catch (error) {
      console.error('Error bulk deleting users:', error);
      return { success: false, error: error instanceof Error ? error.message : 'Failed to connect to server' };
    }
  },

  // Export users
  exportUsers: async (format: 'csv' | 'excel' = 'csv'): Promise<{ success: boolean; error?: string }> => {
    try {
      await usersService.exportUsers(format, usersState.filters);
      return { success: true };
    } catch (error) {
      console.error('Error exporting users:', error);
      return { success: false, error: error instanceof Error ? error.message : 'Failed to connect to server' };
    }
  },
};

export {
  usersState,
  usersActions,
  listLoading,
  createLoading,
  updateLoading,
  deleteLoading,
};