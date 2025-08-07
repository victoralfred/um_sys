import { Component, createSignal, Show } from 'solid-js';
import { useUsers } from '../../contexts/UsersContext';
import { useUI } from '../../contexts/UIContext';
import { useForm } from '../../hooks/useForm';
import { authValidators } from '../../utils/validation';
import { Button } from '../buttons/Button';
import { Input } from '../ui/Input';
import { Card } from '../cards/Card';
import type { CreateUserRequest } from '../../types/user';

interface CreateUserFormData extends Record<string, string> {
  email: string;
  username: string;
  password: string;
  firstName: string;
  lastName: string;
  phoneNumber: string;
}

export const CreateUserModal: Component = () => {
  const users = useUsers();
  const ui = useUI();
  const [isLoading, setIsLoading] = createSignal(false);

  const [formState, formActions] = useForm<CreateUserFormData>(
    ['email', 'username', 'password', 'firstName', 'lastName', 'phoneNumber'],
    {
      initialValues: {
        email: '',
        username: '',
        password: '',
        firstName: '',
        lastName: '',
        phoneNumber: '',
      },
      validator: authValidators.register,
      validateOnChange: false,
      validateOnBlur: true,
    }
  );

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    
    if (!formActions.validate()) {
      ui.notifyError('Validation Error', 'Please fix the errors below');
      return;
    }

    setIsLoading(true);

    try {
      const values = formActions.getValues();
      
      // Convert to CreateUserRequest (remove empty optional fields)
      const createData: CreateUserRequest = {
        email: values.email,
        username: values.username,
        password: values.password,
        ...(values.firstName && { firstName: values.firstName }),
        ...(values.lastName && { lastName: values.lastName }),
        ...(values.phoneNumber && { phoneNumber: values.phoneNumber }),
      };

      const result = await users.createUser(createData);

      if (result.success) {
        ui.notifySuccess('User Created', 'User has been created successfully');
        users.hideCreateModal();
        formActions.reset();
      } else {
        ui.notifyError('Creation Failed', result.error || 'Failed to create user');
        
        // Handle specific errors
        if (result.error?.includes('email')) {
          formActions.setError('email', ['This email is already registered']);
        } else if (result.error?.includes('username')) {
          formActions.setError('username', ['This username is already taken']);
        }
      }
    } catch {
      ui.notifyError('Creation Error', 'An unexpected error occurred');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancel = () => {
    users.hideCreateModal();
    formActions.reset();
  };

  const generatePassword = () => {
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
    const length = 12;
    let password = '';
    
    // Ensure at least one of each required type
    password += 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'[Math.floor(Math.random() * 26)]; // uppercase
    password += 'abcdefghijklmnopqrstuvwxyz'[Math.floor(Math.random() * 26)]; // lowercase
    password += '0123456789'[Math.floor(Math.random() * 10)]; // number
    password += '!@#$%^&*'[Math.floor(Math.random() * 8)]; // special
    
    // Fill remaining characters
    for (let i = 4; i < length; i++) {
      password += chars[Math.floor(Math.random() * chars.length)];
    }
    
    // Shuffle the password
    password = password.split('').sort(() => Math.random() - 0.5).join('');
    
    formActions.setValue('password', password);
    ui.notifyInfo('Password Generated', 'A secure password has been generated');
  };

  return (
    <Show when={users.state.showCreateModal}>
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
              <h2 class="text-heading-lg">Create New User</h2>
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

                <Input
                  label="Phone Number (Optional)"
                  type="tel"
                  placeholder="Enter phone number"
                  value={formState().phoneNumber.value}
                  error={formState().phoneNumber.touched && formState().phoneNumber.errors.length > 0 ? formState().phoneNumber.errors[0] : undefined}
                  onInput={(e) => formActions.setValue('phoneNumber', e.currentTarget.value)}
                  onBlur={() => formActions.setTouched('phoneNumber', true)}
                  autocomplete="tel"
                />
              </div>

              {/* Account Information */}
              <div class="mb-4">
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

                <div class="mb-3">
                  <div class="flex gap-2 items-end">
                    <div class="flex-1">
                      <Input
                        label="Password"
                        type="password"
                        placeholder="Enter password"
                        value={formState().password.value}
                        error={formState().password.touched && formState().password.errors.length > 0 ? formState().password.errors[0] : undefined}
                        onInput={(e) => formActions.setValue('password', e.currentTarget.value)}
                        onBlur={() => formActions.setTouched('password', true)}
                        required
                        autocomplete="new-password"
                        helperText="At least 8 characters with uppercase, lowercase, number, and special character"
                      />
                    </div>
                    <Button
                      variant="secondary"
                      size="sm"
                      type="button"
                      onClick={generatePassword}
                      disabled={isLoading()}
                    >
                      Generate
                    </Button>
                  </div>
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
                  {isLoading() ? 'Creating...' : 'Create User'}
                </Button>
              </div>
            </form>
          </div>
        </Card>
      </div>
    </Show>
  );
};