import { Component, createSignal, Show, createEffect } from 'solid-js';
import { useUsers } from '../../contexts/UsersContext';
import { useUI } from '../../contexts/UIContext';
import { useForm } from '../../hooks/useForm';
import { FormValidator } from '../../utils/validation';
import { Button } from '../buttons/Button';
import { Input } from '../ui/Input';
import { Card } from '../cards/Card';
import type { UpdateUserRequest } from '../../types/user';

interface EditUserFormData extends Record<string, string> {
  email: string;
  username: string;
  firstName: string;
  lastName: string;
  phoneNumber: string;
  bio: string;
}

// Validation rules for user editing
const editUserValidator = new FormValidator({
  email: [
    { rule: 'required', message: 'Email is required' },
    { rule: 'email', message: 'Please enter a valid email address' },
  ],
  username: [
    { rule: 'required', message: 'Username is required' },
    { rule: 'minLength', value: 3, message: 'Username must be at least 3 characters' },
    { rule: 'maxLength', value: 30, message: 'Username must be less than 30 characters' },
    { rule: 'pattern', value: /^[a-zA-Z0-9_-]+$/, message: 'Username can only contain letters, numbers, underscores, and hyphens' },
  ],
  firstName: [
    { rule: 'maxLength', value: 50, message: 'First name must be less than 50 characters' },
  ],
  lastName: [
    { rule: 'maxLength', value: 50, message: 'Last name must be less than 50 characters' },
  ],
  phoneNumber: [
    { rule: 'pattern', value: /^[+]?[\d\s()-]+$/, message: 'Please enter a valid phone number' },
  ],
  bio: [
    { rule: 'maxLength', value: 500, message: 'Bio must be less than 500 characters' },
  ],
});

export const EditUserModal: Component = () => {
  const users = useUsers();
  const ui = useUI();
  const [isLoading, setIsLoading] = createSignal(false);

  const [formState, formActions] = useForm<EditUserFormData>(
    ['email', 'username', 'firstName', 'lastName', 'phoneNumber', 'bio'],
    {
      initialValues: {
        email: '',
        username: '',
        firstName: '',
        lastName: '',
        phoneNumber: '',
        bio: '',
      },
      validator: editUserValidator,
      validateOnChange: false,
      validateOnBlur: true,
    }
  );

  // Load user data when modal opens
  createEffect(() => {
    const user = users.state.selectedUser;
    if (user && users.state.showEditModal) {
      Object.entries({
        email: user.email,
        username: user.username,
        firstName: user.firstName || '',
        lastName: user.lastName || '',
        phoneNumber: user.phoneNumber || '',
        bio: user.bio || '',
      }).forEach(([key, value]) => formActions.setValue(key, value || ''));
    }
  });

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    
    if (!formActions.validate()) {
      ui.notifyError('Validation Error', 'Please fix the errors below');
      return;
    }

    const user = users.state.selectedUser;
    if (!user) return;

    setIsLoading(true);

    try {
      const values = formActions.getValues();
      
      // Build update data (only include changed fields)
      const updateData: UpdateUserRequest = {};
      
      if (values.email !== user.email) updateData.email = values.email;
      if (values.username !== user.username) updateData.username = values.username;
      if (values.firstName !== (user.firstName || '')) updateData.firstName = values.firstName;
      if (values.lastName !== (user.lastName || '')) updateData.lastName = values.lastName;
      if (values.phoneNumber !== (user.phoneNumber || '')) updateData.phoneNumber = values.phoneNumber;
      if (values.bio !== (user.bio || '')) updateData.bio = values.bio;

      // Only proceed if there are changes
      if (Object.keys(updateData).length === 0) {
        ui.notifyInfo('No Changes', 'No changes were made to the user');
        users.hideEditModal();
        return;
      }

      const result = await users.updateUser(user.id, updateData);

      if (result.success) {
        ui.notifySuccess('User Updated', 'User has been updated successfully');
        users.hideEditModal();
      } else {
        ui.notifyError('Update Failed', result.error || 'Failed to update user');
        
        // Handle specific errors
        if (result.error?.includes('email')) {
          formActions.setError('email', ['This email is already registered']);
        } else if (result.error?.includes('username')) {
          formActions.setError('username', ['This username is already taken']);
        }
      }
    } catch {
      ui.notifyError('Update Error', 'An unexpected error occurred');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancel = () => {
    users.hideEditModal();
  };

  const user = () => users.state.selectedUser;

  return (
    <Show when={users.state.showEditModal && user()}>
      <div 
        class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"
        onClick={handleCancel}
        style={{ "backdrop-filter": "blur(2px)" }}
      >
        <Card 
          class="w-full max-w-lg"
          onClick={(e) => e.stopPropagation()}
        >
          <div class="p-6">
            <div class="flex justify-between items-center mb-6">
              <div>
                <h2 class="text-heading-lg">Edit User</h2>
                <p style={{ color: "#6B778C", "font-size": "14px" }}>
                  Editing: {user()!.firstName} {user()!.lastName} (@{user()!.username})
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

            <form onSubmit={handleSubmit}>
              {/* Personal Information */}
              <div class="mb-4">
                <h3 class="text-body-lg mb-3">Personal Information</h3>
                <div class="flex gap-3 mb-3">
                  <Input
                    label="First Name"
                    placeholder="Enter first name"
                    value={formState().firstName.value}
                    error={formState().firstName.touched && formState().firstName.errors.length > 0 ? formState().firstName.errors[0] : undefined}
                    onInput={(e) => formActions.setValue('firstName', e.currentTarget.value)}
                    onBlur={() => formActions.setTouched('firstName', true)}
                    autocomplete="given-name"
                  />
                  <Input
                    label="Last Name"
                    placeholder="Enter last name"
                    value={formState().lastName.value}
                    error={formState().lastName.touched && formState().lastName.errors.length > 0 ? formState().lastName.errors[0] : undefined}
                    onInput={(e) => formActions.setValue('lastName', e.currentTarget.value)}
                    onBlur={() => formActions.setTouched('lastName', true)}
                    autocomplete="family-name"
                  />
                </div>

                <div class="mb-3">
                  <Input
                    label="Phone Number"
                    type="tel"
                    placeholder="Enter phone number"
                    value={formState().phoneNumber.value}
                    error={formState().phoneNumber.touched && formState().phoneNumber.errors.length > 0 ? formState().phoneNumber.errors[0] : undefined}
                    onInput={(e) => formActions.setValue('phoneNumber', e.currentTarget.value)}
                    onBlur={() => formActions.setTouched('phoneNumber', true)}
                    autocomplete="tel"
                  />
                </div>

                <div class="mb-3">
                  <label class="form-label">Bio</label>
                  <textarea
                    placeholder="Enter user bio (optional)"
                    value={formState().bio.value}
                    onInput={(e) => formActions.setValue('bio', e.currentTarget.value)}
                    onBlur={() => formActions.setTouched('bio', true)}
                    class="form-input"
                    style={{ height: "80px", resize: "vertical" }}
                  />
                  <Show when={formState().bio.touched && formState().bio.errors.length > 0}>
                    <div class="text-danger mt-1" style={{ "font-size": "12px" }}>
                      {formState().bio.errors[0]}
                    </div>
                  </Show>
                </div>
              </div>

              {/* Account Information */}
              <div class="mb-6">
                <h3 class="text-body-lg mb-3">Account Information</h3>
                
                <div class="mb-3">
                  <Input
                    label="Email Address"
                    type="email"
                    placeholder="Enter email address"
                    value={formState().email.value}
                    error={formState().email.touched && formState().email.errors.length > 0 ? formState().email.errors[0] : undefined}
                    onInput={(e) => formActions.setValue('email', e.currentTarget.value)}
                    onBlur={() => formActions.setTouched('email', true)}
                    required
                    autocomplete="email"
                  />
                </div>

                <div class="mb-3">
                  <Input
                    label="Username"
                    placeholder="Enter username"
                    value={formState().username.value}
                    error={formState().username.touched && formState().username.errors.length > 0 ? formState().username.errors[0] : undefined}
                    onInput={(e) => formActions.setValue('username', e.currentTarget.value)}
                    onBlur={() => formActions.setTouched('username', true)}
                    required
                    autocomplete="username"
                    helperText="3-30 characters, letters, numbers, underscores, and hyphens only"
                  />
                </div>
              </div>

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
                  variant="primary"
                  type="submit"
                  loading={isLoading()}
                  disabled={isLoading()}
                >
                  {isLoading() ? 'Updating...' : 'Update User'}
                </Button>
              </div>
            </form>
          </div>
        </Card>
      </div>
    </Show>
  );
};