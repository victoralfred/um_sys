import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { usersState, usersActions } from '../users';
import { usersService } from '../../services/users';
import type { User, UserListResponse } from '../../types/user';

// Mock the users service
vi.mock('../../services/users', () => ({
  usersService: {
    getUsers: vi.fn(),
    createUser: vi.fn(),
    updateUser: vi.fn(),
    deleteUser: vi.fn(),
    getUser: vi.fn(),
    bulkDeleteUsers: vi.fn(),
    exportUsers: vi.fn(),
  },
}));

describe('Users Store', () => {
  const mockUser: User = {
    id: '1',
    email: 'test@example.com',
    username: 'testuser',
    firstName: 'Test',
    lastName: 'User',
    status: 'active',
    emailVerified: true,
    phoneVerified: false,
    mfaEnabled: false,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  };

  const mockUserListResponse: UserListResponse = {
    success: true,
    data: {
      users: [mockUser],
      total: 1,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    },
  };

  beforeEach(() => {
    // Clear all mocks before each test
    vi.clearAllMocks();
    
    // Reset users state
    usersState.users = [];
    usersState.selectedUser = null;
    usersState.loading = false;
    usersState.error = null;
    usersState.pagination = {
      page: 1,
      pageSize: 20,
      total: 0,
      totalPages: 0,
    };
    usersState.filters = {
      search: '',
      sortBy: 'createdAt',
      sortOrder: 'desc',
    };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Initial State', () => {
    it('should have correct initial state', () => {
      expect(usersState.users).toEqual([]);
      expect(usersState.selectedUser).toBeNull();
      expect(usersState.loading).toBe(false);
      expect(usersState.error).toBeNull();
      expect(usersState.pagination.page).toBe(1);
      expect(usersState.pagination.pageSize).toBe(20);
      expect(usersState.filters.sortBy).toBe('createdAt');
      expect(usersState.filters.sortOrder).toBe('desc');
    });
  });

  describe('Load Users', () => {
    it('should load users successfully', async () => {
      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      await usersActions.loadUsers();

      expect(vi.mocked(usersService).getUsers).toHaveBeenCalled();
      expect(usersState.users).toEqual([mockUser]);
      expect(usersState.pagination.total).toBe(1);
      expect(usersState.pagination.totalPages).toBe(1);
      expect(usersState.loading).toBe(false);
      expect(usersState.error).toBeNull();
    });

    it('should handle load users failure', async () => {
      const errorResponse = {
        success: false,
        error: {
          code: 'SERVER_ERROR',
          message: 'Internal server error',
        },
      };

      vi.mocked(usersService).getUsers.mockResolvedValue(errorResponse);

      await usersActions.loadUsers();

      expect(usersState.users).toEqual([]);
      expect(usersState.error).toBe('Internal server error');
      expect(usersState.loading).toBe(false);
    });

    it('should handle network errors', async () => {
      vi.mocked(usersService).getUsers.mockRejectedValue(new Error('Network error'));

      await usersActions.loadUsers();

      expect(usersState.error).toBe('Failed to connect to server');
      expect(usersState.loading).toBe(false);
    });

    it('should merge filters when loading users', async () => {
      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      await usersActions.loadUsers({ search: 'test', emailVerified: true });

      expect(vi.mocked(usersService).getUsers).toHaveBeenCalledWith({
        search: 'test',
        emailVerified: true,
        sortBy: 'createdAt',
        sortOrder: 'desc',
      });
    });
  });

  describe('Create User', () => {
    it('should create user successfully', async () => {
      const newUserData = {
        email: 'new@example.com',
        username: 'newuser',
        password: 'password123',
        firstName: 'New',
        lastName: 'User',
      };

      const createResponse = {
        success: true,
        data: { ...mockUser, id: '2', email: 'new@example.com' },
      };

      vi.mocked(usersService).createUser.mockResolvedValue(createResponse);
      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      const result = await usersActions.createUser(newUserData);

      expect(result.success).toBe(true);
      expect(vi.mocked(usersService).createUser).toHaveBeenCalledWith(newUserData);
      expect(vi.mocked(usersService).getUsers).toHaveBeenCalled(); // Should reload users
      expect(usersState.showCreateModal).toBe(false);
    });

    it('should handle create user failure', async () => {
      const newUserData = {
        email: 'existing@example.com',
        username: 'existinguser',
        password: 'password123',
      };

      const errorResponse = {
        success: false,
        error: {
          code: 'EMAIL_EXISTS',
          message: 'Email already exists',
        },
      };

      vi.mocked(usersService).createUser.mockResolvedValue(errorResponse);

      const result = await usersActions.createUser(newUserData);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Email already exists');
      expect(usersState.showCreateModal).toBe(true); // Should remain open
    });
  });

  describe('Update User', () => {
    it('should update user successfully', async () => {
      // Set initial users state
      usersState.users = [mockUser];

      const updates = {
        firstName: 'Updated',
        lastName: 'Name',
      };

      const updateResponse = {
        success: true,
        data: { ...mockUser, ...updates },
      };

      vi.mocked(usersService).updateUser.mockResolvedValue(updateResponse);

      const result = await usersActions.updateUser('1', updates);

      expect(result.success).toBe(true);
      expect(vi.mocked(usersService).updateUser).toHaveBeenCalledWith('1', updates);
      expect(usersState.users[0].firstName).toBe('Updated');
      expect(usersState.users[0].lastName).toBe('Name');
      expect(usersState.showEditModal).toBe(false);
    });

    it('should update selected user if it matches', async () => {
      usersState.users = [mockUser];
      usersState.selectedUser = mockUser;

      const updates = { firstName: 'Updated' };
      const updateResponse = {
        success: true,
        data: { ...mockUser, ...updates },
      };

      vi.mocked(usersService).updateUser.mockResolvedValue(updateResponse);

      await usersActions.updateUser('1', updates);

      expect(usersState.selectedUser?.firstName).toBe('Updated');
    });

    it('should handle update user failure', async () => {
      const errorResponse = {
        success: false,
        error: {
          code: 'VALIDATION_ERROR',
          message: 'Invalid data',
        },
      };

      vi.mocked(usersService).updateUser.mockResolvedValue(errorResponse);

      const result = await usersActions.updateUser('1', { firstName: '' });

      expect(result.success).toBe(false);
      expect(result.error).toBe('Invalid data');
    });
  });

  describe('Delete User', () => {
    it('should delete user successfully', async () => {
      // Set initial users state
      usersState.users = [mockUser, { ...mockUser, id: '2' }];

      const deleteResponse = { success: true };
      vi.mocked(usersService).deleteUser.mockResolvedValue(deleteResponse);

      const result = await usersActions.deleteUser('1');

      expect(result.success).toBe(true);
      expect(vi.mocked(usersService).deleteUser).toHaveBeenCalledWith('1');
      expect(usersState.users).toHaveLength(1);
      expect(usersState.users[0].id).toBe('2');
      expect(usersState.showDeleteModal).toBe(false);
    });

    it('should clear selected user if it was deleted', async () => {
      usersState.users = [mockUser];
      usersState.selectedUser = mockUser;

      const deleteResponse = { success: true };
      vi.mocked(usersService).deleteUser.mockResolvedValue(deleteResponse);

      await usersActions.deleteUser('1');

      expect(usersState.selectedUser).toBeNull();
    });

    it('should handle delete user failure', async () => {
      const errorResponse = {
        success: false,
        error: {
          code: 'PERMISSION_DENIED',
          message: 'Cannot delete this user',
        },
      };

      vi.mocked(usersService).deleteUser.mockResolvedValue(errorResponse);

      const result = await usersActions.deleteUser('1');

      expect(result.success).toBe(false);
      expect(result.error).toBe('Cannot delete this user');
    });
  });

  describe('Get Single User', () => {
    it('should get user successfully', async () => {
      const getUserResponse = {
        success: true,
        data: mockUser,
      };

      vi.mocked(usersService).getUser.mockResolvedValue(getUserResponse);

      const result = await usersActions.getUser('1');

      expect(result).toEqual(mockUser);
      expect(vi.mocked(usersService).getUser).toHaveBeenCalledWith('1');
    });

    it('should return null if user not found', async () => {
      const getUserResponse = {
        success: false,
        error: { code: 'NOT_FOUND', message: 'User not found' },
      };

      vi.mocked(usersService).getUser.mockResolvedValue(getUserResponse);

      const result = await usersActions.getUser('999');

      expect(result).toBeNull();
    });
  });

  describe('User Selection', () => {
    it('should select user', () => {
      usersActions.selectUser(mockUser);
      expect(usersState.selectedUser).toEqual(mockUser);
    });

    it('should clear selection', () => {
      usersState.selectedUser = mockUser;
      usersActions.selectUser(null);
      expect(usersState.selectedUser).toBeNull();
    });
  });

  describe('Filtering and Pagination', () => {
    it('should set filters and reload users', async () => {
      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      await usersActions.setFilters({ search: 'test', emailVerified: true });

      expect(usersState.filters.search).toBe('test');
      expect(usersState.filters.emailVerified).toBe(true);
      expect(usersState.pagination.page).toBe(1); // Should reset to first page
      expect(vi.mocked(usersService).getUsers).toHaveBeenCalled();
    });

    it('should set sorting and reload users', async () => {
      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      await usersActions.setSorting('email', 'asc');

      expect(usersState.sortBy).toBe('email');
      expect(usersState.sortOrder).toBe('asc');
      expect(usersState.filters.sortBy).toBe('email');
      expect(usersState.filters.sortOrder).toBe('asc');
      expect(vi.mocked(usersService).getUsers).toHaveBeenCalled();
    });

    it('should set pagination and reload users', async () => {
      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      await usersActions.setPagination(2, 50);

      expect(usersState.pagination.page).toBe(2);
      expect(usersState.pagination.pageSize).toBe(50);
      expect(usersState.filters.offset).toBe(50); // (2-1) * 50
      expect(usersState.filters.limit).toBe(50);
      expect(vi.mocked(usersService).getUsers).toHaveBeenCalled();
    });

    it('should search users', async () => {
      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      await usersActions.searchUsers('test query');

      expect(usersState.filters.search).toBe('test query');
      expect(usersState.pagination.page).toBe(1); // Should reset to first page
      expect(vi.mocked(usersService).getUsers).toHaveBeenCalled();
    });

    it('should clear filters', async () => {
      // Set some filters first
      usersState.filters = {
        search: 'test',
        emailVerified: true,
        sortBy: 'email',
        sortOrder: 'asc',
      };

      vi.mocked(usersService.getUsers).mockResolvedValue(mockUserListResponse);

      await usersActions.clearFilters();

      expect(usersState.filters.search).toBe('');
      expect(usersState.filters.emailVerified).toBeUndefined();
      expect(usersState.filters.sortBy).toBe('createdAt');
      expect(usersState.filters.sortOrder).toBe('desc');
      expect(usersState.pagination.page).toBe(1);
      expect(vi.mocked(usersService).getUsers).toHaveBeenCalled();
    });
  });

  describe('Modal Actions', () => {
    it('should show and hide create modal', () => {
      usersActions.showCreateModal();
      expect(usersState.showCreateModal).toBe(true);

      usersActions.hideCreateModal();
      expect(usersState.showCreateModal).toBe(false);
    });

    it('should show and hide edit modal', () => {
      usersActions.showEditModal(mockUser);
      expect(usersState.showEditModal).toBe(true);
      expect(usersState.selectedUser).toEqual(mockUser);

      usersActions.hideEditModal();
      expect(usersState.showEditModal).toBe(false);
    });

    it('should show and hide delete modal', () => {
      usersActions.showDeleteModal(mockUser);
      expect(usersState.showDeleteModal).toBe(true);
      expect(usersState.selectedUser).toEqual(mockUser);

      usersActions.hideDeleteModal();
      expect(usersState.showDeleteModal).toBe(false);
    });
  });

  describe('Bulk Operations', () => {
    it('should bulk delete users successfully', async () => {
      usersState.users = [
        mockUser,
        { ...mockUser, id: '2' },
        { ...mockUser, id: '3' },
      ];

      const bulkDeleteResponse = { success: true };
      vi.mocked(usersService).bulkDeleteUsers.mockResolvedValue(bulkDeleteResponse);

      const result = await usersActions.bulkDelete(['1', '2']);

      expect(result.success).toBe(true);
      expect(vi.mocked(usersService).bulkDeleteUsers).toHaveBeenCalledWith(['1', '2']);
      expect(usersState.users).toHaveLength(1);
      expect(usersState.users[0].id).toBe('3');
    });

    it('should handle bulk delete failure', async () => {
      const bulkDeleteResponse = {
        success: false,
        error: { message: 'Some users could not be deleted' },
      };
      vi.mocked(usersService).bulkDeleteUsers.mockResolvedValue(bulkDeleteResponse);

      const result = await usersActions.bulkDelete(['1', '2']);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Some users could not be deleted');
    });

    it('should export users successfully', async () => {
      vi.mocked(usersService).exportUsers.mockResolvedValue(undefined);

      const result = await usersActions.exportUsers('csv');

      expect(result.success).toBe(true);
      expect(vi.mocked(usersService).exportUsers).toHaveBeenCalledWith('csv', usersState.filters);
    });

    it('should handle export failure', async () => {
      vi.mocked(usersService).exportUsers.mockRejectedValue(new Error('Export failed'));

      const result = await usersActions.exportUsers('excel');

      expect(result.success).toBe(false);
      expect(result.error).toBe('Export failed');
    });
  });
});