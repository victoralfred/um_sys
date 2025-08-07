import { Component, createSignal, Show } from 'solid-js';
import { useUsers } from '../../contexts/UsersContext';
import { useUI } from '../../contexts/UIContext';
import { useAuth } from '../../contexts/AuthContext';
import { Button } from '../buttons/Button';
import { Card } from '../cards/Card';
import { Badge } from '../ui/Badge';

export const DeleteUserModal: Component = () => {
  const users = useUsers();
  const ui = useUI();
  const auth = useAuth();
  const [isLoading, setIsLoading] = createSignal(false);
  const [confirmText, setConfirmText] = createSignal('');

  const handleDelete = async () => {
    const user = users.state.selectedUser;
    if (!user) return;

    // Prevent self-deletion
    if (user.id === auth.state.user?.id) {
      ui.notifyWarning('Cannot Delete', 'You cannot delete your own account');
      return;
    }

    // Check confirmation text
    const expectedText = `${user.username}`;
    if (confirmText() !== expectedText) {
      ui.notifyWarning('Confirmation Required', `Please type "${expectedText}" to confirm deletion`);
      return;
    }

    setIsLoading(true);

    try {
      const result = await users.deleteUser(user.id);

      if (result.success) {
        ui.notifySuccess('User Deleted', `User "${user.username}" has been deleted successfully`);
        users.hideDeleteModal();
        setConfirmText('');
      } else {
        ui.notifyError('Delete Failed', result.error || 'Failed to delete user');
      }
    } catch {
      ui.notifyError('Delete Error', 'An unexpected error occurred');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancel = () => {
    users.hideDeleteModal();
    setConfirmText('');
  };

  const user = () => users.state.selectedUser;
  const isCurrentUser = () => user()?.id === auth.state.user?.id;
  const confirmationText = () => user() ? user()!.username : '';
  const isConfirmationValid = () => confirmText() === confirmationText();

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
    <Show when={users.state.showDeleteModal && user()}>
      <div 
        class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"
        onClick={handleCancel}
        style={{ "backdrop-filter": "blur(2px)" }}
      >
        <Card 
          class="w-full max-w-md"
          onClick={(e) => e.stopPropagation()}
        >
          <div class="p-6">
            <div class="flex justify-between items-start mb-6">
              <div>
                <h2 class="text-heading-lg text-danger mb-2">Delete User</h2>
                <p style={{ color: "#6B778C", "font-size": "14px" }}>
                  This action cannot be undone
                </p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleCancel}
                disabled={isLoading()}
              >
                ×
              </Button>
            </div>

            {/* User Information */}
            <div class="mb-6 p-4 border rounded-lg" style={{ "border-color": "#DFE1E6", "background-color": "#F7F8F9" }}>
              <div class="flex items-center gap-3 mb-2">
                <div class="w-10 h-10 rounded-full flex items-center justify-center" style={{ "background-color": "#0052CC", color: "white" }}>
                  {user()!.firstName?.[0] || user()!.username[0].toUpperCase()}
                </div>
                <div>
                  <div class="font-semibold">
                    {user()!.firstName} {user()!.lastName}
                  </div>
                  <div style={{ color: "#6B778C", "font-size": "12px" }}>
                    @{user()!.username}
                  </div>
                </div>
              </div>
              
              <div class="flex items-center gap-2 mb-2">
                <span style={{ "font-size": "12px", color: "#6B778C" }}>Email:</span>
                <span style={{ "font-size": "12px" }}>{user()!.email}</span>
              </div>

              <div class="flex items-center gap-2">
                <span style={{ "font-size": "12px", color: "#6B778C" }}>Status:</span>
                <Badge variant={getStatusBadgeVariant(user()!.status)} size="sm">
                  {user()!.status}
                </Badge>
                <Show when={user()!.emailVerified}>
                  <Badge variant="success" size="sm">Email ✓</Badge>
                </Show>
                <Show when={user()!.mfaEnabled}>
                  <Badge variant="info" size="sm">2FA</Badge>
                </Show>
              </div>
            </div>

            {/* Self-deletion warning */}
            <Show when={isCurrentUser()}>
              <div class="mb-6 p-4 border rounded-lg" style={{ "border-color": "#FFBDAD", "background-color": "#FFF5F5" }}>
                <div class="flex items-center gap-2 text-danger mb-2">
                  <span>⚠️</span>
                  <span class="font-semibold">Cannot Delete Own Account</span>
                </div>
                <p style={{ "font-size": "14px", color: "#6B778C" }}>
                  You cannot delete your own account. Please ask another administrator to delete your account if needed.
                </p>
              </div>
            </Show>

            {/* Warning message */}
            <Show when={!isCurrentUser()}>
              <div class="mb-6 p-4 border rounded-lg" style={{ "border-color": "#FFBDAD", "background-color": "#FFF5F5" }}>
                <div class="flex items-center gap-2 text-danger mb-2">
                  <span>⚠️</span>
                  <span class="font-semibold">Permanent Deletion</span>
                </div>
                <ul style={{ "font-size": "14px", color: "#6B778C", "padding-left": "16px" }}>
                  <li>• All user data will be permanently deleted</li>
                  <li>• The user will lose access to their account</li>
                  <li>• This action cannot be reversed</li>
                  <li>• User history and associated data will be removed</li>
                </ul>
              </div>

              {/* Confirmation Input */}
              <div class="mb-6">
                <label class="form-label">
                  Type <strong>{confirmationText()}</strong> to confirm deletion:
                </label>
                <input
                  type="text"
                  placeholder={`Type "${confirmationText()}" here`}
                  value={confirmText()}
                  onInput={(e) => setConfirmText(e.currentTarget.value)}
                  class="form-input mt-2"
                  autocomplete="off"
                  disabled={isLoading()}
                />
                <Show when={confirmText() && !isConfirmationValid()}>
                  <div class="text-danger mt-1" style={{ "font-size": "12px" }}>
                    Text does not match. Please type "{confirmationText()}" exactly.
                  </div>
                </Show>
              </div>
            </Show>

            {/* Actions */}
            <div class="flex gap-3 justify-end">
              <Button
                variant="secondary"
                onClick={handleCancel}
                disabled={isLoading()}
              >
                Cancel
              </Button>
              <Button
                variant="danger"
                onClick={handleDelete}
                loading={isLoading()}
                disabled={isLoading() || isCurrentUser() || !isConfirmationValid()}
              >
                {isLoading() ? 'Deleting...' : 'Delete User'}
              </Button>
            </div>
          </div>
        </Card>
      </div>
    </Show>
  );
};