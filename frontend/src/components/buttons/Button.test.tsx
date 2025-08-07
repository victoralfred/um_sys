import { render, screen } from '@solidjs/testing-library';
import { describe, test, expect, vi } from 'vitest';
import { Button } from './Button';

describe('Button', () => {
  test('renders with default primary variant', () => {
    render(() => <Button>Click me</Button>);
    
    const button = screen.getByRole('button', { name: 'Click me' });
    expect(button).toBeInTheDocument();
    expect(button).toHaveClass('btn', 'btn-primary');
  });

  test('renders secondary variant correctly', () => {
    render(() => <Button variant="secondary">Secondary</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('btn-secondary');
    expect(button).not.toHaveClass('btn-primary');
  });

  test('renders danger variant correctly', () => {
    render(() => <Button variant="danger">Delete</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('btn-danger');
  });

  test('renders ghost variant correctly', () => {
    render(() => <Button variant="ghost">Ghost</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('btn-ghost');
  });

  test('applies size classes correctly', () => {
    render(() => <Button size="sm">Small</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('btn-sm');
  });

  test('shows loading state correctly', () => {
    render(() => <Button loading>Loading</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveClass('btn-loading');
    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
  });

  test('handles disabled state correctly', () => {
    render(() => <Button disabled>Disabled</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('disabled');
  });

  test('calls onClick handler when clicked', async () => {
    const handleClick = vi.fn();
    render(() => <Button onClick={handleClick}>Click me</Button>);
    
    const button = screen.getByRole('button');
    button.click();
    
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  test('does not call onClick when disabled', async () => {
    const handleClick = vi.fn();
    render(() => <Button disabled onClick={handleClick}>Click me</Button>);
    
    const button = screen.getByRole('button');
    button.click();
    
    expect(handleClick).not.toHaveBeenCalled();
  });

  test('applies fullWidth class correctly', () => {
    render(() => <Button fullWidth>Full Width</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('w-full');
  });

  test('supports custom class names', () => {
    render(() => <Button class="custom-class">Custom</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('custom-class');
  });

  test('has proper accessibility attributes', () => {
    render(() => <Button aria-label="Custom label">Button</Button>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-label', 'Custom label');
    expect(button).toHaveAttribute('type', 'button');
  });
});