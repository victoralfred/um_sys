import { Component, createSignal, Show } from 'solid-js';
import { useAuth } from '../../contexts/AuthContext';
import { useUI } from '../../contexts/UIContext';
import { useForm } from '../../hooks/useForm';
import { authValidators } from '../../utils/validation';
import { Button } from '../../components/buttons/Button';
import { Input } from '../../components/ui/Input';
import { Card } from '../../components/cards/Card';

interface LoginFormData extends Record<string, string> {
  email: string;
  password: string;
}

export const LoginPage: Component = () => {
  const auth = useAuth();
  const ui = useUI();
  const [isLoading, setIsLoading] = createSignal(false);
  const [rememberMe, setRememberMe] = createSignal(false);

  const [formState, formActions] = useForm<LoginFormData>(
    ['email', 'password'],
    {
      initialValues: { email: '', password: '' },
      validator: authValidators.login,
      validateOnChange: false, // Validate only on blur for better UX
      validateOnBlur: true,
    }
  );

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    
    // Validate form before submission
    if (!formActions.validate()) {
      ui.notifyError('Validation Error', 'Please fix the errors below');
      return;
    }

    setIsLoading(true);
    
    try {
      const values = formActions.getValues();
      const result = await auth.login(values);

      if (result.success) {
        ui.notifySuccess('Welcome Back!', 'You have successfully logged in');
      } else {
        ui.notifyError(
          'Login Failed', 
          result.error?.message || 'Invalid email or password'
        );
        
        // Focus email field if invalid credentials
        if (result.error?.code === 'INVALID_CREDENTIALS') {
          document.getElementById('email')?.focus();
        }
      }
    } catch {
      ui.notifyError('Login Error', 'An unexpected error occurred. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleForgotPassword = () => {
    const email = formState().email.value;
    if (email && authValidators.login.validateField('email', email, {}).isValid) {
      ui.notifyInfo(
        'Password Reset', 
        `Password reset instructions will be sent to ${email}`
      );
    } else {
      ui.notifyWarning(
        'Email Required', 
        'Please enter a valid email address first'
      );
      document.getElementById('email')?.focus();
    }
  };

  const handleDemoLogin = async () => {
    formActions.setValue('email', 'demo@example.com');
    formActions.setValue('password', 'Demo123!@#');
    
    // Auto-submit after setting demo values
    if (typeof window !== 'undefined') {
      window.setTimeout(() => {
        const form = document.getElementById('login-form') as HTMLFormElement | null;
        form?.requestSubmit();
      }, 100);
    }
  };

  return (
    <div class="min-h-screen flex items-center justify-center p-4" style={{ "background-color": "#F7F8F9" }}>
      <Card class="w-full" style={{ "max-width": "400px" }}>
        <div class="p-6">
          {/* Header */}
          <div class="text-center mb-6">
            <h1 class="text-heading-lg mb-2">Welcome Back</h1>
            <p style={{ color: "#6B778C" }}>
              Sign in to your UManager account
            </p>
          </div>

          {/* Login Form */}
          <form id="login-form" onSubmit={handleSubmit}>
            <div class="mb-4">
              <Input
                id="email"
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

            <div class="mb-4">
              <Input
                id="password"
                label="Password"
                type="password"
                placeholder="Enter your password"
                value={formState().password.value}
                error={formState().password.touched && formState().password.errors.length > 0 ? formState().password.errors[0] : undefined}
                onInput={(e) => formActions.setValue('password', e.currentTarget.value)}
                onBlur={() => formActions.setTouched('password', true)}
                required
                autocomplete="current-password"
              />
            </div>

            {/* Remember Me & Forgot Password */}
            <div class="flex justify-between items-center mb-6">
              <label class="flex items-center gap-2">
                <input 
                  type="checkbox" 
                  checked={rememberMe()}
                  onChange={(e) => setRememberMe(e.currentTarget.checked)}
                />
                <span style={{ "font-size": "14px" }}>Remember me</span>
              </label>
              
              <Button
                variant="ghost"
                size="sm"
                type="button"
                onClick={handleForgotPassword}
              >
                Forgot Password?
              </Button>
            </div>

            {/* Submit Button */}
            <Button
              variant="primary"
              type="submit"
              loading={isLoading()}
              disabled={isLoading() || !formState().email.value || !formState().password.value}
              class="w-full mb-4"
            >
              {isLoading() ? 'Signing In...' : 'Sign In'}
            </Button>
          </form>

          {/* Demo Login */}
          <div class="mb-4">
            <div class="relative">
              <hr style={{ "border-color": "#DFE1E6" }} />
              <span 
                class="absolute left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2 px-2"
                style={{ "background-color": "white", color: "#6B778C", "font-size": "12px" }}
              >
                OR
              </span>
            </div>
          </div>

          <Button
            variant="secondary"
            type="button"
            onClick={handleDemoLogin}
            disabled={isLoading()}
            class="w-full mb-4"
          >
            Try Demo Account
          </Button>

          {/* Register Link */}
          <div class="text-center">
            <p style={{ "font-size": "14px", color: "#6B778C" }}>
              Don't have an account?{' '}
              <a href="/register">
                <Button variant="ghost" size="sm" type="button">
                  Create Account
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
              <p>Authenticating...</p>
            </div>
          </Card>
        </div>
      </Show>
    </div>
  );
};