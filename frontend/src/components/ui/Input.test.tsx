import { render, screen, fireEvent } from '@solidjs/testing-library';
import { describe, test, expect, vi } from 'vitest';
import { createSignal } from 'solid-js';
import { Input } from './Input';

describe('Input', () => {
  test('renders with default type text', () => {
    render(() => <Input placeholder="Enter text" />);
    
    const input = screen.getByPlaceholderText('Enter text');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('type', 'text');
    expect(input).toHaveClass('form-input');
  });

  test('renders different input types correctly', () => {
    render(() => <Input type="email" placeholder="Email" />);
    
    const input = screen.getByPlaceholderText('Email');
    expect(input).toHaveAttribute('type', 'email');
  });

  test('renders password type with visibility toggle', () => {
    render(() => <Input type="password" placeholder="Password" showPasswordToggle />);
    
    const input = screen.getByPlaceholderText('Password');
    const toggleButton = screen.getByRole('button');
    
    expect(input).toHaveAttribute('type', 'password');
    expect(toggleButton).toBeInTheDocument();
  });

  test('toggles password visibility when button clicked', async () => {
    render(() => <Input type="password" placeholder="Password" showPasswordToggle />);
    
    const input = screen.getByPlaceholderText('Password') as HTMLInputElement;
    const toggleButton = screen.getByRole('button');
    
    expect(input.type).toBe('password');
    
    fireEvent.click(toggleButton);
    expect(input.type).toBe('text');
    
    fireEvent.click(toggleButton);
    expect(input.type).toBe('password');
  });

  test('shows error state correctly', () => {
    render(() => <Input error="This field is required" placeholder="Input" />);
    
    const input = screen.getByPlaceholderText('Input');
    const errorMsg = screen.getByText('This field is required');
    
    expect(input).toHaveClass('form-input--error');
    expect(errorMsg).toBeInTheDocument();
    expect(errorMsg).toHaveClass('text-ds-danger-bold');
  });

  test('shows label when provided', () => {
    render(() => <Input label="Username" placeholder="Enter username" />);
    
    const label = screen.getByText('Username');
    const input = screen.getByPlaceholderText('Enter username');
    
    expect(label).toBeInTheDocument();
    expect(label).toHaveClass('form-label');
    expect(input).toHaveAttribute('id');
    expect(label).toHaveAttribute('for', input.id);
  });

  test('shows required indicator when required', () => {
    render(() => <Input label="Email" required placeholder="Email" />);
    
    const label = screen.getByText('Email');
    expect(label.textContent).toContain('*');
  });

  test('handles disabled state correctly', () => {
    render(() => <Input disabled placeholder="Disabled input" />);
    
    const input = screen.getByPlaceholderText('Disabled input');
    expect(input).toBeDisabled();
    expect(input).toHaveClass('form-input');
  });

  test('calls onChange handler when value changes', async () => {
    const handleChange = vi.fn();
    const [value, setValue] = createSignal('');
    
    render(() => <Input value={value()} onChange={(val) => { setValue(val); handleChange(val); }} placeholder="Input" />);
    
    const input = screen.getByPlaceholderText('Input') as HTMLInputElement;
    
    fireEvent.input(input, { target: { value: 'test' } });
    
    expect(handleChange).toHaveBeenCalledWith('test');
    expect(value()).toBe('test');
  });

  test('supports controlled input with value prop', () => {
    render(() => <Input value="controlled value" placeholder="Input" />);
    
    const input = screen.getByPlaceholderText('Input') as HTMLInputElement;
    expect(input.value).toBe('controlled value');
  });

  test('shows helper text when provided', () => {
    render(() => <Input helperText="This is helper text" placeholder="Input" />);
    
    const helperText = screen.getByText('This is helper text');
    expect(helperText).toBeInTheDocument();
    expect(helperText).toHaveClass('text-ds-text-subtle');
  });

  test('has proper accessibility attributes', () => {
    render(() => (
      <Input 
        label="Email" 
        placeholder="Enter email" 
        required 
        aria-describedby="email-help"
        helperText="We'll never share your email"
      />
    ));
    
    const input = screen.getByPlaceholderText('Enter email');
    const label = screen.getByText('Email');
    
    expect(input).toHaveAttribute('required');
    expect(input).toHaveAttribute('aria-describedby');
    expect(label).toHaveAttribute('for', input.id);
  });
});