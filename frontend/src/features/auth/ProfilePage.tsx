import { Component, createSignal, Show } from 'solid-js';
import { useAuth } from '../../contexts/AuthContext';
import { useUI } from '../../contexts/UIContext';
import { useForm } from '../../hooks/useForm';
import { authValidators } from '../../utils/validation';
import { Button } from '../../components/buttons/Button';
import { Input } from '../../components/ui/Input';
import { Card } from '../../components/cards/Card';
import { Badge } from '../../components/ui/Badge';

interface ProfileFormData extends Record<string, string> {
  email: string;
  username: string;
  firstName: string;
  lastName: string;
  phoneNumber: string;
}

interface PasswordFormData extends Record<string, string> {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

export const ProfilePage: Component = () => {
  const auth = useAuth();
  const ui = useUI();
  const [isEditingProfile, setIsEditingProfile] = createSignal(false);
  const [isChangingPassword, setIsChangingPassword] = createSignal(false);
  const [profileLoading, setProfileLoading] = createSignal(false);
  const [passwordLoading, setPasswordLoading] = createSignal(false);

  const user = () => auth.state.user;

  // Profile form
  const [profileState, profileActions] = useForm<ProfileFormData>(
    ['email', 'username', 'firstName', 'lastName', 'phoneNumber'],
    {
      initialValues: {
        email: user()?.email || '',
        username: user()?.username || '',
        firstName: user()?.firstName || '',
        lastName: user()?.lastName || '',
        phoneNumber: '', // TODO: Add phone number to user type
      },
      validator: authValidators.profile,
      validateOnBlur: true,
    }
  );

  // Password form
  const [passwordState, passwordActions] = useForm<PasswordFormData>(
    ['currentPassword', 'newPassword', 'confirmPassword'],
    {
      initialValues: {
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      },
      validator: authValidators.changePassword,
      validateOnBlur: true,
    }
  );

  const handleProfileUpdate = async (e: Event) => {
    e.preventDefault();

    if (!profileActions.validate()) {
      ui.notifyError('Validation Error', 'Please fix the errors below');
      return;
    }

    setProfileLoading(true);

    try {
      const values = profileActions.getValues();
      // TODO: Call API to update profile
      
      // Simulate API call
      await new Promise(resolve => {
        if (typeof window !== 'undefined') {
          window.setTimeout(resolve, 1000);
        } else {
          resolve(undefined);
        }
      });
      
      // Update auth state
      auth.updateProfile(values);
      
      ui.notifySuccess('Profile Updated', 'Your profile has been updated successfully');
      setIsEditingProfile(false);
    } catch {
      ui.notifyError('Update Failed', 'Failed to update profile. Please try again.');
    } finally {
      setProfileLoading(false);
    }
  };

  const handlePasswordChange = async (e: Event) => {
    e.preventDefault();

    if (!passwordActions.validate()) {
      ui.notifyError('Validation Error', 'Please fix the errors below');
      return;
    }

    setPasswordLoading(true);

    try {
      passwordActions.getValues();
      // TODO: Call API to change password with values
      
      // Simulate API call
      await new Promise(resolve => {
        if (typeof window !== 'undefined') {
          window.setTimeout(resolve, 1500);
        } else {
          resolve(undefined);
        }
      });
      
      ui.notifySuccess('Password Changed', 'Your password has been updated successfully');
      setIsChangingPassword(false);
      passwordActions.reset();
    } catch {
      ui.notifyError('Password Change Failed', 'Failed to change password. Please try again.');
    } finally {
      setPasswordLoading(false);
    }
  };

  const handleCancelEdit = () => {
    setIsEditingProfile(false);
    // Reset form to original values
    profileActions.reset({
      email: user()?.email || '',
      username: user()?.username || '',
      firstName: user()?.firstName || '',
      lastName: user()?.lastName || '',
      phoneNumber: '',
    });
  };

  const handleCancelPasswordChange = () => {
    setIsChangingPassword(false);
    passwordActions.reset();
  };

  const handleLogout = async () => {
    ui.openModal(
      'div',
      {
        children: (
          <div class="p-6 text-center">
            <h3 class="text-heading-sm mb-4">Confirm Logout</h3>
            <p class="mb-6">Are you sure you want to logout?</p>
            <div class="flex gap-3 justify-center">
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
                  await auth.logout();
                  ui.notifyInfo('Logged Out', 'You have been successfully logged out');
                }}
              >
                Logout
              </Button>
            </div>
          </div>
        ),
      },
      { size: 'sm' }
    );
  };

  return (
    <div class="container-ds py-8">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-heading-lg mb-2">Profile Settings</h1>
          <p style={{ color: "#6B778C" }}>
            Manage your account information and preferences
          </p>
        </div>
        <Button variant="danger" onClick={handleLogout}>
          Logout
        </Button>
      </div>

      <div class="grid gap-6" style={{ "grid-template-columns": "1fr 1fr" }}>
        {/* Profile Information */}
        <Card>
          <div class="p-6">
            <div class="flex justify-between items-center mb-6">
              <h2 class="text-heading-md">Profile Information</h2>
              <Show 
                when={!isEditingProfile()} 
                fallback={
                  <div class="flex gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleCancelEdit}
                      disabled={profileLoading()}
                    >
                      Cancel
                    </Button>
                  </div>
                }
              >
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setIsEditingProfile(true)}
                >
                  Edit Profile
                </Button>
              </Show>
            </div>

            <Show 
              when={isEditingProfile()}
              fallback={
                <div class="space-y-4">
                  <div>
                    <label class="form-label">Full Name</label>
                    <p class="text-body">
                      {user()?.firstName} {user()?.lastName}
                    </p>
                  </div>
                  <div>
                    <label class="form-label">Username</label>
                    <p class="text-body">@{user()?.username}</p>
                  </div>
                  <div>
                    <label class="form-label">Email Address</label>
                    <div class="flex items-center gap-2">
                      <p class="text-body">{user()?.email}</p>
                      <Badge variant="success" size="sm">Verified</Badge>
                    </div>
                  </div>
                  <div>
                    <label class="form-label">Account Status</label>
                    <Badge variant="success">Active</Badge>
                  </div>
                </div>
              }
            >
              <form onSubmit={handleProfileUpdate}>
                <div class="grid gap-4">
                  <div class="grid gap-3" style={{ "grid-template-columns": "1fr 1fr" }}>
                    <Input
                      label="First Name"
                      value={profileState().firstName.value}
                      error={profileState().firstName.touched && profileState().firstName.errors.length > 0 ? profileState().firstName.errors[0] : undefined}
                      onInput={(e) => profileActions.setValue('firstName', e.currentTarget.value)}
                      onBlur={() => profileActions.setTouched('firstName', true)}
                    />
                    <Input
                      label="Last Name"
                      value={profileState().lastName.value}
                      error={profileState().lastName.touched && profileState().lastName.errors.length > 0 ? profileState().lastName.errors[0] : undefined}
                      onInput={(e) => profileActions.setValue('lastName', e.currentTarget.value)}
                      onBlur={() => profileActions.setTouched('lastName', true)}
                    />
                  </div>
                  
                  <Input
                    label="Username"
                    value={profileState().username.value}
                    error={profileState().username.touched && profileState().username.errors.length > 0 ? profileState().username.errors[0] : undefined}
                    onInput={(e) => profileActions.setValue('username', e.currentTarget.value)}
                    onBlur={() => profileActions.setTouched('username', true)}
                    required
                  />

                  <Input
                    label="Email Address"
                    type="email"
                    value={profileState().email.value}
                    error={profileState().email.touched && profileState().email.errors.length > 0 ? profileState().email.errors[0] : undefined}
                    onInput={(e) => profileActions.setValue('email', e.currentTarget.value)}
                    onBlur={() => profileActions.setTouched('email', true)}
                    required
                  />

                  <Input
                    label="Phone Number"
                    type="tel"
                    placeholder="Optional"
                    value={profileState().phoneNumber.value}
                    error={profileState().phoneNumber.touched && profileState().phoneNumber.errors.length > 0 ? profileState().phoneNumber.errors[0] : undefined}
                    onInput={(e) => profileActions.setValue('phoneNumber', e.currentTarget.value)}
                    onBlur={() => profileActions.setTouched('phoneNumber', true)}
                  />

                  <Button
                    variant="primary"
                    type="submit"
                    loading={profileLoading()}
                    disabled={profileLoading()}
                    class="w-full"
                  >
                    {profileLoading() ? 'Updating...' : 'Update Profile'}
                  </Button>
                </div>
              </form>
            </Show>
          </div>
        </Card>

        {/* Security Settings */}
        <Card>
          <div class="p-6">
            <div class="flex justify-between items-center mb-6">
              <h2 class="text-heading-md">Security Settings</h2>
            </div>

            {/* Password Section */}
            <div class="mb-6">
              <div class="flex justify-between items-center mb-4">
                <div>
                  <h3 class="text-body-lg">Password</h3>
                  <p style={{ color: "#6B778C", "font-size": "14px" }}>
                    Last changed 30 days ago
                  </p>
                </div>
                <Show when={!isChangingPassword()}>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setIsChangingPassword(true)}
                  >
                    Change Password
                  </Button>
                </Show>
              </div>

              <Show when={isChangingPassword()}>
                <form onSubmit={handlePasswordChange}>
                  <div class="grid gap-4">
                    <Input
                      label="Current Password"
                      type="password"
                      value={passwordState().currentPassword.value}
                      error={passwordState().currentPassword.touched && passwordState().currentPassword.errors.length > 0 ? passwordState().currentPassword.errors[0] : undefined}
                      onInput={(e) => passwordActions.setValue('currentPassword', e.currentTarget.value)}
                      onBlur={() => passwordActions.setTouched('currentPassword', true)}
                      required
                    />

                    <Input
                      label="New Password"
                      type="password"
                      value={passwordState().newPassword.value}
                      error={passwordState().newPassword.touched && passwordState().newPassword.errors.length > 0 ? passwordState().newPassword.errors[0] : undefined}
                      onInput={(e) => passwordActions.setValue('newPassword', e.currentTarget.value)}
                      onBlur={() => passwordActions.setTouched('newPassword', true)}
                      required
                      helperText="At least 8 characters with uppercase, lowercase, number, and special character"
                    />

                    <Input
                      label="Confirm New Password"
                      type="password"
                      value={passwordState().confirmPassword.value}
                      error={passwordState().confirmPassword.touched && passwordState().confirmPassword.errors.length > 0 ? passwordState().confirmPassword.errors[0] : undefined}
                      onInput={(e) => passwordActions.setValue('confirmPassword', e.currentTarget.value)}
                      onBlur={() => passwordActions.setTouched('confirmPassword', true)}
                      required
                    />

                    <div class="flex gap-2">
                      <Button
                        variant="primary"
                        type="submit"
                        loading={passwordLoading()}
                        disabled={passwordLoading()}
                      >
                        {passwordLoading() ? 'Changing...' : 'Change Password'}
                      </Button>
                      <Button
                        variant="ghost"
                        type="button"
                        onClick={handleCancelPasswordChange}
                        disabled={passwordLoading()}
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>
                </form>
              </Show>
            </div>

            {/* Two-Factor Authentication */}
            <div class="mb-6">
              <div class="flex justify-between items-center">
                <div>
                  <h3 class="text-body-lg">Two-Factor Authentication</h3>
                  <p style={{ color: "#6B778C", "font-size": "14px" }}>
                    Add an extra layer of security to your account
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <Badge variant="neutral">Disabled</Badge>
                  <Button variant="secondary" size="sm">
                    Enable
                  </Button>
                </div>
              </div>
            </div>

            {/* Active Sessions */}
            <div>
              <div class="flex justify-between items-center mb-4">
                <h3 class="text-body-lg">Active Sessions</h3>
                <Button variant="ghost" size="sm">
                  View All
                </Button>
              </div>
              <div class="space-y-2">
                <div 
                  class="p-3 rounded"
                  style={{ "background-color": "#F7F8F9", border: "1px solid #DFE1E6" }}
                >
                  <div class="flex justify-between items-center">
                    <div>
                      <p class="text-body-sm">Current Session</p>
                      <p style={{ color: "#6B778C", "font-size": "12px" }}>
                        Chrome on Mac • Last active now
                      </p>
                    </div>
                    <Badge variant="success" size="sm">Current</Badge>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
};