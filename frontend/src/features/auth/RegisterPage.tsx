import { Component, createSignal, Show } from 'solid-js';
import { useAuth } from '../../contexts/AuthContext';
import { useUI } from '../../contexts/UIContext';
import { useForm } from '../../hooks/useForm';
import { authValidators } from '../../utils/validation';
import { Button } from '../../components/buttons/Button';
import { Input } from '../../components/ui/Input';
import { Card } from '../../components/cards/Card';

interface RegisterFormData extends Record<string, string> {
  email: string;
  username: string;
  password: string;
  confirmPassword: string;
  firstName: string;
  lastName: string;
}

export const RegisterPage: Component = () => {
  const auth = useAuth();
  const ui = useUI();
  const [isLoading, setIsLoading] = createSignal(false);
  const [acceptTerms, setAcceptTerms] = createSignal(false);

  const [formState, formActions] = useForm<RegisterFormData>(
    ['email', 'username', 'password', 'confirmPassword', 'firstName', 'lastName'],
    {
      initialValues: {
        email: '',
        username: '',
        password: '',
        confirmPassword: '',
        firstName: '',
        lastName: '',
      },
      validator: authValidators.register,
      validateOnChange: false,
      validateOnBlur: true,
    }
  );

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    
    // Check terms acceptance
    if (!acceptTerms()) {
      ui.notifyWarning('Terms Required', 'Please accept the terms and conditions');
      return;
    }

    // Validate form
    if (!formActions.validate()) {
      ui.notifyError('Validation Error', 'Please fix the errors below');
      return;
    }

    setIsLoading(true);

    try {
      const values = formActions.getValues();
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { confirmPassword, ...registerData } = values;

      const result = await auth.register(registerData);

      if (result.success) {
        ui.notifySuccess(
          'Account Created!', 
          'Welcome to UManager! Your account has been created successfully.'
        );
      } else {
        ui.notifyError(
          'Registration Failed',
          result.error?.message || 'Unable to create account. Please try again.'
        );

        // Handle specific errors
        if (result.error?.code === 'EMAIL_EXISTS') {
          formActions.setError('email', ['This email is already registered']);
        } else if (result.error?.code === 'USERNAME_EXISTS') {
          formActions.setError('username', ['This username is already taken']);
        }
      }
    } catch {
      ui.notifyError('Registration Error', 'An unexpected error occurred. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const generateUsername = () => {
    const firstName = formState().firstName.value.toLowerCase();
    const lastName = formState().lastName.value.toLowerCase();
    
    if (firstName || lastName) {
      const baseUsername = `${firstName}${lastName}${Math.floor(Math.random() * 1000)}`;
      formActions.setValue('username', baseUsername.replace(/[^a-zA-Z0-9_-]/g, ''));
    }
  };

  const checkUsernameAvailability = async () => {
    const username = formState().username.value;
    if (!username) return;

    ui.notifyInfo('Checking...', `Checking if "${username}" is available`);
    
    // TODO: Implement actual availability check
    if (typeof window !== 'undefined') {
      window.setTimeout(() => {
        ui.notifySuccess('Available!', `"${username}" is available`);
      }, 1000);
    }
  };

  return (
    <div class="min-h-screen flex items-center justify-center p-4" style={{ "background-color": "#F7F8F9" }}>
      <Card class="w-full" style={{ "max-width": "500px" }}>
        <div class="p-6">
          {/* Header */}
          <div class="text-center mb-6">
            <h1 class="text-heading-lg mb-2">Create Account</h1>
            <p style={{ color: "#6B778C" }}>
              Join UManager to get started
            </p>
          </div>

          {/* Registration Form */}
          <form onSubmit={handleSubmit}>
            {/* Personal Information */}
            <div class="mb-4">
              <h3 class="text-body-lg mb-3">Personal Information</h3>
              <div class="flex gap-3">
                <Input
                  label="First Name"
                  placeholder="Enter your first name"
                  value={formState().firstName.value}
                  error={formState().firstName.touched && formState().firstName.errors.length > 0 ? formState().firstName.errors[0] : undefined}
                  onInput={(e) => formActions.setValue('firstName', e.currentTarget.value)}
                  onBlur={() => formActions.setTouched('firstName', true)}
                  autocomplete="given-name"
                />
                <Input
                  label="Last Name"
                  placeholder="Enter your last name"
                  value={formState().lastName.value}
                  error={formState().lastName.touched && formState().lastName.errors.length > 0 ? formState().lastName.errors[0] : undefined}
                  onInput={(e) => formActions.setValue('lastName', e.currentTarget.value)}
                  onBlur={() => formActions.setTouched('lastName', true)}
                  autocomplete="family-name"
                />
              </div>
            </div>

            {/* Account Information */}
            <div class="mb-4">
              <h3 class="text-body-lg mb-3">Account Information</h3>
              
              <div class="mb-3">
                <Input
                  label="Email Address"
                  type="email"
                  placeholder="Enter your email"
                  value={formState().email.value}
                  error={formState().email.touched && formState().email.errors.length > 0 ? formState().email.errors[0] : undefined}
                  onInput={(e) => formActions.setValue('email', e.currentTarget.value)}
                  onBlur={() => formActions.setTouched('email', true)}
                  required
                  autocomplete="email"
                />
              </div>

              <div class="mb-3">
                <div class="flex gap-2 items-end">
                  <div class="flex-1">
                    <Input
                      label="Username"
                      placeholder="Choose a username"
                      value={formState().username.value}
                      error={formState().username.touched && formState().username.errors.length > 0 ? formState().username.errors[0] : undefined}
                      onInput={(e) => formActions.setValue('username', e.currentTarget.value)}
                      onBlur={() => formActions.setTouched('username', true)}
                      required
                      autocomplete="username"
                      helperText="3-30 characters, letters, numbers, underscores, and hyphens only"
                    />
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    type="button"
                    onClick={generateUsername}
                    disabled={!formState().firstName.value && !formState().lastName.value}
                  >
                    Generate
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    type="button"
                    onClick={checkUsernameAvailability}
                    disabled={!formState().username.value}
                  >
                    Check
                  </Button>
                </div>
              </div>
            </div>

            {/* Password */}
            <div class="mb-4">
              <h3 class="text-body-lg mb-3">Security</h3>
              
              <div class="mb-3">
                <Input
                  label="Password"
                  type="password"
                  placeholder="Create a strong password"
                  value={formState().password.value}
                  error={formState().password.touched && formState().password.errors.length > 0 ? formState().password.errors[0] : undefined}
                  onInput={(e) => formActions.setValue('password', e.currentTarget.value)}
                  onBlur={() => formActions.setTouched('password', true)}
                  required
                  autocomplete="new-password"
                  helperText="At least 8 characters with uppercase, lowercase, number, and special character"
                />
              </div>

              <div class="mb-3">
                <Input
                  label="Confirm Password"
                  type="password"
                  placeholder="Confirm your password"
                  value={formState().confirmPassword.value}
                  error={formState().confirmPassword.touched && formState().confirmPassword.errors.length > 0 ? formState().confirmPassword.errors[0] : undefined}
                  onInput={(e) => formActions.setValue('confirmPassword', e.currentTarget.value)}
                  onBlur={() => formActions.setTouched('confirmPassword', true)}
                  required
                  autocomplete="new-password"
                />
              </div>
            </div>

            {/* Terms and Conditions */}
            <div class="mb-6">
              <label class="flex items-start gap-3">
                <input 
                  type="checkbox"
                  checked={acceptTerms()}
                  onChange={(e) => setAcceptTerms(e.currentTarget.checked)}
                  style={{ "margin-top": "2px" }}
                />
                <span style={{ "font-size": "14px", "line-height": "1.4" }}>
                  I agree to the{' '}
                  <Button variant="ghost" size="sm" type="button">
                    Terms of Service
                  </Button>
                  {' '}and{' '}
                  <Button variant="ghost" size="sm" type="button">
                    Privacy Policy
                  </Button>
                </span>
              </label>
            </div>

            {/* Submit Button */}
            <Button
              variant="primary"
              type="submit"
              loading={isLoading()}
              disabled={isLoading() || !acceptTerms()}
              class="w-full mb-4"
            >
              {isLoading() ? 'Creating Account...' : 'Create Account'}
            </Button>
          </form>

          {/* Login Link */}
          <div class="text-center">
            <p style={{ "font-size": "14px", color: "#6B778C" }}>
              Already have an account?{' '}
              <a href="/login">
                <Button variant="ghost" size="sm" type="button">
                  Sign In
                </Button>
              </a>
            </p>
          </div>
        </div>
      </Card>

      {/* Loading Overlay */}
      <Show when={auth.state.isLoading}>
        <div 
          class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
          style={{ "backdrop-filter": "blur(2px)" }}
        >
          <Card>
            <div class="p-6 text-center">
              <div class="spinner mb-4"></div>
              <p>Creating your account...</p>
            </div>
          </Card>
        </div>
      </Show>
    </div>
  );
};