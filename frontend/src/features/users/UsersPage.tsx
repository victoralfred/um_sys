import { Component, onMount, Show, createSignal } from 'solid-js';
import { useUsers } from '../../contexts/UsersContext';
import { useUI } from '../../contexts/UIContext';
import { useAuth } from '../../contexts/AuthContext';
import { DataTable, TableColumn, TableAction } from '../../components/tables/DataTable';
import { Button } from '../../components/buttons/Button';
import { Badge } from '../../components/ui/Badge';
import { Card } from '../../components/cards/Card';
import { CreateUserModal, EditUserModal, DeleteUserModal } from '../../components/modals';
import type { User, UserStatus } from '../../types/user';

export const UsersPage: Component = () => {
  const users = useUsers();
  const ui = useUI();
  const auth = useAuth();
  
  const [selectedUsers, setSelectedUsers] = createSignal<User[]>([]);
  const [filterStatus, setFilterStatus] = createSignal<UserStatus | ''>('');
  const [filterVerified, setFilterVerified] = createSignal<boolean | ''>('');

  onMount(() => {
    users.loadUsers();
  });

  const getStatusBadgeVariant = (status: UserStatus) => {
    switch (status) {
      case 'active':
        return 'success';
      case 'inactive':
        return 'neutral';
      case 'suspended':
        return 'warning';
      case 'locked':
        return 'danger';
      default:
        return 'neutral';
    }
  };

  const columns: TableColumn<User>[] = [
    {
      key: 'firstName',
      label: 'User',
      sortable: true,
      render: (_, user) => (
        <div>
          <div style={{ "font-weight": "500" }}>
            {user.firstName} {user.lastName}
          </div>
          <div style={{ "font-size": "12px", color: "#6B778C" }}>
            @{user.username}
          </div>
        </div>
      ),
    },
    {
      key: 'email',
      label: 'Email',
      sortable: true,
      render: (email) => (
        <span class="text-body">{email}</span>
      ),
    },
    {
      key: 'status',
      label: 'Status',
      sortable: true,
      render: (status) => (
        <Badge variant={getStatusBadgeVariant(status)}>
          {status}
        </Badge>
      ),
    },
    {
      key: 'emailVerified',
      label: 'Verification',
      render: (_, user) => (
        <div class="flex gap-1">
          <Show when={user.emailVerified}>
            <Badge variant="success" size="sm">Email ✓</Badge>
          </Show>
          <Show when={user.phoneVerified}>
            <Badge variant="success" size="sm">Phone ✓</Badge>
          </Show>
          <Show when={user.mfaEnabled}>
            <Badge variant="info" size="sm">2FA</Badge>
          </Show>
          <Show when={!user.emailVerified && !user.phoneVerified && !user.mfaEnabled}>
            <Badge variant="neutral" size="sm">Unverified</Badge>
          </Show>
        </div>
      ),
    },
    {
      key: 'createdAt',
      label: 'Created',
      sortable: true,
      render: (createdAt) => (
        <span class="text-body-sm" style={{ color: "#6B778C" }}>
          {new Date(createdAt).toLocaleDateString()}
        </span>
      ),
    },
  ];

  const actions: TableAction<User>[] = [
    {
      label: 'Edit',
      variant: 'ghost',
      onClick: (user) => {
        users.showEditModal(user);
      },
    },
    {
      label: 'Delete',
      variant: 'danger',
      onClick: (user) => {
        users.showDeleteModal(user);
      },
      disabled: (user) => user.id === auth.state.user?.id, // Can't delete yourself
    },
  ];

  const bulkActions = [
    {
      label: 'Delete Selected',
      variant: 'danger' as const,
      onClick: (selected: User[]) => {
        // Filter out current user to prevent self-deletion
        const userIds = selected
          .filter(user => user.id !== auth.state.user?.id)
          .map(user => user.id);
        
        if (userIds.length === 0) {
          ui.notifyWarning('Invalid Selection', 'Cannot delete your own account');
          return;
        }

        ui.openModal(
          'div',
          {
            children: (
              <div class="p-6">
                <h3 class="text-heading-sm mb-4">Confirm Bulk Delete</h3>
                <p class="mb-6">
                  Are you sure you want to delete {userIds.length} user{userIds.length > 1 ? 's' : ''}? 
                  This action cannot be undone.
                </p>
                <div class="flex gap-3 justify-end">
                  <Button
                    variant="secondary"
                    onClick={() => ui.closeAllModals()}
                  >
                    Cancel
                  </Button>
                  <Button
                    variant="danger"
                    onClick={async () => {
                      ui.closeAllModals();
                      const result = await users.bulkDelete(userIds);
                      if (result.success) {
                        ui.notifySuccess('Users Deleted', `${userIds.length} user${userIds.length > 1 ? 's' : ''} deleted successfully`);
                        setSelectedUsers([]);
                      } else {
                        ui.notifyError('Delete Failed', result.error || 'Failed to delete users');
                      }
                    }}
                  >
                    Delete {userIds.length} User{userIds.length > 1 ? 's' : ''}
                  </Button>
                </div>
              </div>
            ),
          },
          { size: 'sm' }
        );
      },
    },
    {
      label: 'Export Selected',
      variant: 'secondary' as const,
      onClick: (selected: User[]) => {
        ui.notifyInfo('Export Started', `Exporting ${selected.length} users...`);
        // TODO: Implement export selected users
      },
    },
  ];

  const handleSearch = (query: string) => {
    users.searchUsers(query);
  };

  const handleSort = (column: string | number | symbol, direction: 'asc' | 'desc') => {
    users.setSorting(String(column), direction);
  };

  const handlePageChange = (page: number) => {
    users.setPagination(page);
  };

  const handlePageSizeChange = (pageSize: number) => {
    users.setPagination(users.state.pagination.page, pageSize);
  };

  const applyFilters = () => {
    const filters: Record<string, string | boolean> = {};
    
    if (filterStatus()) {
      filters.status = filterStatus();
    }
    
    if (filterVerified() !== '') {
      filters.emailVerified = filterVerified();
    }
    
    users.setFilters(filters);
  };

  const clearFilters = () => {
    setFilterStatus('');
    setFilterVerified('');
    users.clearFilters();
  };

  return (
    <div class="p-8">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-heading-lg mb-2">User Management</h1>
          <p style={{ color: "#6B778C" }}>
            Manage user accounts, permissions, and settings
          </p>
        </div>

        <div class="flex gap-3">
          <Button
            variant="secondary"
            onClick={() => users.exportUsers('csv')}
            disabled={users.loading.list()}
          >
            Export CSV
          </Button>
          <Button
            variant="secondary"
            onClick={() => users.exportUsers('excel')}
            disabled={users.loading.list()}
          >
            Export Excel
          </Button>
          <Button
            variant="primary"
            onClick={() => users.showCreateModal()}
          >
            Create User
          </Button>
        </div>
      </div>

      {/* Filters */}
      <Card class="mb-6">
        <div class="p-4">
          <div class="flex gap-4 items-end">
            <div>
              <label class="form-label">Status</label>
              <select
                value={filterStatus()}
                onChange={(e) => setFilterStatus(e.currentTarget.value as UserStatus | '')}
                class="form-input"
                style={{ width: "150px" }}
              >
                <option value="">All Statuses</option>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
                <option value="suspended">Suspended</option>
                <option value="locked">Locked</option>
              </select>
            </div>

            <div>
              <label class="form-label">Email Verified</label>
              <select
                value={String(filterVerified())}
                onChange={(e) => setFilterVerified(
                  e.currentTarget.value === '' ? '' : e.currentTarget.value === 'true'
                )}
                class="form-input"
                style={{ width: "150px" }}
              >
                <option value="">All Users</option>
                <option value="true">Verified Only</option>
                <option value="false">Unverified Only</option>
              </select>
            </div>

            <Button variant="secondary" onClick={applyFilters}>
              Apply Filters
            </Button>

            <Button variant="ghost" onClick={clearFilters}>
              Clear Filters
            </Button>
          </div>
        </div>
      </Card>

      {/* Users Table */}
      <DataTable
        data={users.state.users}
        columns={columns}
        loading={users.loading.list()}
        error={users.state.error || undefined}
        actions={actions}
        pagination={users.state.pagination}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
        onSort={handleSort}
        sortColumn={users.state.sortBy}
        sortDirection={users.state.sortOrder}
        selectable={true}
        selectedRows={selectedUsers()}
        onSelectionChange={setSelectedUsers}
        searchable={true}
        onSearch={handleSearch}
        searchQuery={users.state.filters.search}
        bulkActions={bulkActions}
        emptyMessage="No users found. Create your first user to get started."
      />

      {/* Stats Card */}
      <Show when={users.state.users.length > 0}>
        <div class="grid gap-4 mt-6" style={{ "grid-template-columns": "repeat(auto-fit, minmax(200px, 1fr))" }}>
          <Card>
            <div class="p-4 text-center">
              <div class="text-heading-md mb-1">
                {users.state.pagination.total}
              </div>
              <div style={{ color: "#6B778C", "font-size": "14px" }}>
                Total Users
              </div>
            </div>
          </Card>

          <Card>
            <div class="p-4 text-center">
              <div class="text-heading-md mb-1">
                {users.state.users.filter(u => u.status === 'active').length}
              </div>
              <div style={{ color: "#6B778C", "font-size": "14px" }}>
                Active Users
              </div>
            </div>
          </Card>

          <Card>
            <div class="p-4 text-center">
              <div class="text-heading-md mb-1">
                {users.state.users.filter(u => u.emailVerified).length}
              </div>
              <div style={{ color: "#6B778C", "font-size": "14px" }}>
                Verified Users
              </div>
            </div>
          </Card>

          <Card>
            <div class="p-4 text-center">
              <div class="text-heading-md mb-1">
                {users.state.users.filter(u => u.mfaEnabled).length}
              </div>
              <div style={{ color: "#6B778C", "font-size": "14px" }}>
                2FA Enabled
              </div>
            </div>
          </Card>
        </div>
      </Show>

      {/* Modals */}
      <CreateUserModal />
      <EditUserModal />
      <DeleteUserModal />
    </div>
  );
};