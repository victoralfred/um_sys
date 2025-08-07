import { Component, createSignal, For, Show, onMount } from 'solid-js';
import { useUsers } from '../contexts/UsersContext';
import { useUI } from '../contexts/UIContext';
import { useAuth } from '../contexts/AuthContext';
import { Button } from '../components/buttons/Button';
import { Input } from '../components/ui/Input';
import { Badge } from '../components/ui/Badge';
import { Card } from '../components/cards/Card';
import type { User } from '../types/user';

// Example component demonstrating user management
export const UsersExample: Component = () => {
  const users = useUsers();
  const ui = useUI();
  const auth = useAuth();
  
  const [searchQuery, setSearchQuery] = createSignal('');
  const [selectedUsers, setSelectedUsers] = createSignal<string[]>([]);

  onMount(() => {
    // Load users when component mounts
    users.loadUsers();
  });

  const handleSearch = () => {
    users.searchUsers(searchQuery());
  };

  const handleCreateUser = () => {
    ui.notifyInfo('Create User', 'This would open a create user modal');
    // In a real app, this would open a modal with a form
    // users.showCreateModal();
  };

  const handleEditUser = (user: User) => {
    ui.notifyInfo('Edit User', `This would edit ${user.username}`);
    // users.showEditModal(user);
  };

  const handleDeleteUser = (user: User) => {
    ui.notifyInfo('Delete User', `This would delete ${user.username}`);
    // users.showDeleteModal(user);
  };

  const handleBulkDelete = () => {
    const selected = selectedUsers();
    if (selected.length === 0) {
      ui.notifyWarning('No Selection', 'Please select users to delete');
      return;
    }
    
    ui.notifyInfo('Bulk Delete', `This would delete ${selected.length} users`);
    // users.bulkDelete(selected);
    setSelectedUsers([]);
  };

  const handleExport = (format: 'csv' | 'excel') => {
    ui.notifyInfo('Export Users', `This would export users as ${format.toUpperCase()}`);
    // users.exportUsers(format);
  };

  const toggleUserSelection = (userId: string) => {
    setSelectedUsers(prev => 
      prev.includes(userId) 
        ? prev.filter(id => id !== userId)
        : [...prev, userId]
    );
  };

  const getStatusBadgeVariant = (status: string) => {
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

  return (
    <div style={{ "max-width": "1200px", margin: "0 auto", padding: "20px" }}>
      <div class="flex justify-between items-center mb-4">
        <h1>User Management</h1>
        <Show when={auth.state.isAuthenticated}>
          <Button variant="primary" onClick={handleCreateUser}>
            Create User
          </Button>
        </Show>
      </div>

      {/* Search and Filters */}
      <Card class="mb-4">
        <div class="p-4">
          <div class="flex gap-2 items-end">
            <div class="flex-1">
              <Input
                label="Search Users"
                placeholder="Search by name, email, or username..."
                value={searchQuery()}
                onInput={(e) => setSearchQuery(e.currentTarget.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    handleSearch();
                  }
                }}
              />
            </div>
            <Button variant="secondary" onClick={handleSearch}>
              Search
            </Button>
            <Button variant="ghost" onClick={() => {
              setSearchQuery('');
              users.clearFilters();
            }}>
              Clear
            </Button>
          </div>

          <div class="flex gap-2 mt-4">
            <Button 
              variant="secondary" 
              size="sm"
              onClick={() => handleExport('csv')}
            >
              Export CSV
            </Button>
            <Button 
              variant="secondary" 
              size="sm"
              onClick={() => handleExport('excel')}
            >
              Export Excel
            </Button>
            <Show when={selectedUsers().length > 0}>
              <Button 
                variant="danger" 
                size="sm"
                onClick={handleBulkDelete}
              >
                Delete Selected ({selectedUsers().length})
              </Button>
            </Show>
          </div>
        </div>
      </Card>

      {/* Loading State */}
      <Show when={users.loading.list()}>
        <div class="text-center p-4">
          <div class="spinner"></div>
          <p class="mt-2">Loading users...</p>
        </div>
      </Show>

      {/* Error State */}
      <Show when={users.state.error}>
        <div class="card p-4 mb-4" style={{ "border-color": "#DE350B", "background-color": "#FFF4F1" }}>
          <p style={{ color: "#DE350B" }}>
            <strong>Error:</strong> {users.state.error}
          </p>
          <Button variant="secondary" size="sm" class="mt-2" onClick={() => users.loadUsers()}>
            Retry
          </Button>
        </div>
      </Show>

      {/* Users Table */}
      <Show when={!users.loading.list() && users.state.users.length > 0}>
        <Card>
          <div style={{ overflow: "auto" }}>
            <table style={{ width: "100%", "border-collapse": "collapse" }}>
              <thead>
                <tr style={{ "border-bottom": "1px solid #DFE1E6" }}>
                  <th style={{ padding: "12px", "text-align": "left" }}>
                    <input 
                      type="checkbox"
                      onChange={(e) => {
                        if (e.currentTarget.checked) {
                          setSelectedUsers(users.state.users.map(u => u.id));
                        } else {
                          setSelectedUsers([]);
                        }
                      }}
                    />
                  </th>
                  <th style={{ padding: "12px", "text-align": "left" }}>User</th>
                  <th style={{ padding: "12px", "text-align": "left" }}>Email</th>
                  <th style={{ padding: "12px", "text-align": "left" }}>Status</th>
                  <th style={{ padding: "12px", "text-align": "left" }}>Verification</th>
                  <th style={{ padding: "12px", "text-align": "left" }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                <For each={users.state.users}>
                  {(user) => (
                    <tr style={{ "border-bottom": "1px solid #F1F2F4" }}>
                      <td style={{ padding: "12px" }}>
                        <input 
                          type="checkbox"
                          checked={selectedUsers().includes(user.id)}
                          onChange={() => toggleUserSelection(user.id)}
                        />
                      </td>
                      <td style={{ padding: "12px" }}>
                        <div>
                          <div style={{ "font-weight": "500" }}>
                            {user.firstName} {user.lastName}
                          </div>
                          <div style={{ "font-size": "12px", color: "#6B778C" }}>
                            @{user.username}
                          </div>
                        </div>
                      </td>
                      <td style={{ padding: "12px" }}>{user.email}</td>
                      <td style={{ padding: "12px" }}>
                        <Badge variant={getStatusBadgeVariant(user.status)}>
                          {user.status}
                        </Badge>
                      </td>
                      <td style={{ padding: "12px" }}>
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
                        </div>
                      </td>
                      <td style={{ padding: "12px" }}>
                        <div class="flex gap-1">
                          <Button 
                            variant="ghost" 
                            size="sm"
                            onClick={() => handleEditUser(user)}
                          >
                            Edit
                          </Button>
                          <Button 
                            variant="danger" 
                            size="sm"
                            onClick={() => handleDeleteUser(user)}
                          >
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  )}
                </For>
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          <div class="flex justify-between items-center p-4" style={{ "border-top": "1px solid #F1F2F4" }}>
            <div style={{ "font-size": "14px", color: "#6B778C" }}>
              Showing {users.state.users.length} of {users.state.pagination.total} users
            </div>
            <div class="flex gap-2">
              <Button 
                variant="secondary" 
                size="sm"
                disabled={users.state.pagination.page <= 1}
                onClick={() => users.setPagination(users.state.pagination.page - 1)}
              >
                Previous
              </Button>
              <span class="flex items-center px-3" style={{ "font-size": "14px" }}>
                Page {users.state.pagination.page} of {users.state.pagination.totalPages}
              </span>
              <Button 
                variant="secondary" 
                size="sm"
                disabled={users.state.pagination.page >= users.state.pagination.totalPages}
                onClick={() => users.setPagination(users.state.pagination.page + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        </Card>
      </Show>

      {/* Empty State */}
      <Show when={!users.loading.list() && users.state.users.length === 0}>
        <div class="text-center p-8">
          <h3>No users found</h3>
          <p style={{ color: "#6B778C" }}>
            {users.state.filters.search ? 'Try adjusting your search criteria' : 'Create your first user to get started'}
          </p>
          <Show when={!users.state.filters.search}>
            <Button variant="primary" class="mt-4" onClick={handleCreateUser}>
              Create First User
            </Button>
          </Show>
        </div>
      </Show>
    </div>
  );
};