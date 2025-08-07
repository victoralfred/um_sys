import { Component, createSignal, Show } from 'solid-js';
import { useAuth } from '../contexts/AuthContext';
import { useUI } from '../contexts/UIContext';
import { Button } from '../components/buttons/Button';
import { Input } from '../components/ui/Input';

// Example component demonstrating authentication flow
export const AuthExample: Component = () => {
  const auth = useAuth();
  const ui = useUI();
  
  const [email, setEmail] = createSignal('');
  const [password, setPassword] = createSignal('');
  const [isRegistering, setIsRegistering] = createSignal(false);
  const [firstName, setFirstName] = createSignal('');
  const [lastName, setLastName] = createSignal('');
  const [username, setUsername] = createSignal('');

  const handleLogin = async (e: Event) => {
    e.preventDefault();
    
    const result = await auth.login({
      email: email(),
      password: password(),
    });

    if (result.success) {
      ui.notifySuccess('Login Successful', 'Welcome back!');
    } else {
      ui.notifyError('Login Failed', result.error?.message || 'Please try again');
    }
  };

  const handleRegister = async (e: Event) => {
    e.preventDefault();
    
    const result = await auth.register({
      email: email(),
      username: username(),
      password: password(),
      firstName: firstName(),
      lastName: lastName(),
    });

    if (result.success) {
      ui.notifySuccess('Registration Successful', 'Welcome to UManager!');
    } else {
      ui.notifyError('Registration Failed', result.error?.message || 'Please try again');
    }
  };

  const handleLogout = async () => {
    await auth.logout();
    ui.notifyInfo('Logged Out', 'See you next time!');
    // Clear form
    setEmail('');
    setPassword('');
    setFirstName('');
    setLastName('');
    setUsername('');
  };

  return (
    <div class="card p-4" style={{ "max-width": "400px", margin: "0 auto" }}>
      <Show 
        when={auth.state.isAuthenticated} 
        fallback={
          <div>
            <h2 class="text-center mb-4">
              {isRegistering() ? 'Create Account' : 'Sign In'}
            </h2>
            
            <form onSubmit={isRegistering() ? handleRegister : handleLogin}>
              <div class="mb-4">
                <Input
                  label="Email"
                  type="email"
                  value={email()}
                  onInput={(e) => setEmail(e.currentTarget.value)}
                  required
                />
              </div>

              <Show when={isRegistering()}>
                <div class="mb-4">
                  <Input
                    label="Username"
                    type="text"
                    value={username()}
                    onInput={(e) => setUsername(e.currentTarget.value)}
                    required
                  />
                </div>

                <div class="flex gap-2 mb-4">
                  <Input
                    label="First Name"
                    type="text"
                    value={firstName()}
                    onInput={(e) => setFirstName(e.currentTarget.value)}
                  />
                  <Input
                    label="Last Name"
                    type="text"
                    value={lastName()}
                    onInput={(e) => setLastName(e.currentTarget.value)}
                  />
                </div>
              </Show>

              <div class="mb-4">
                <Input
                  label="Password"
                  type="password"
                  value={password()}
                  onInput={(e) => setPassword(e.currentTarget.value)}
                  required
                />
              </div>

              <Button
                variant="primary"
                type="submit"
                loading={auth.state.isLoading}
                class="w-full mb-4"
              >
                {isRegistering() ? 'Create Account' : 'Sign In'}
              </Button>

              <Button
                variant="ghost"
                type="button"
                onClick={() => setIsRegistering(!isRegistering())}
                class="w-full"
              >
                {isRegistering() ? 'Already have an account? Sign In' : "Don't have an account? Sign Up"}
              </Button>
            </form>
          </div>
        }
      >
        <div class="text-center">
          <h2 class="mb-4">Welcome, {auth.state.user?.firstName || auth.state.user?.username}!</h2>
          
          <div class="mb-4 p-4" style={{ "background-color": "#f7f8f9", "border-radius": "4px" }}>
            <h3 class="mb-2">User Information</h3>
            <p><strong>Email:</strong> {auth.state.user?.email}</p>
            <p><strong>Username:</strong> {auth.state.user?.username}</p>
            <p><strong>Name:</strong> {auth.state.user?.firstName} {auth.state.user?.lastName}</p>
          </div>

          <Button
            variant="secondary"
            onClick={handleLogout}
            class="w-full"
          >
            Sign Out
          </Button>
        </div>
      </Show>
    </div>
  );
};