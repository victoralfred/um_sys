import type { Component, JSX } from 'solid-js';
import { mergeProps, Show, splitProps } from 'solid-js';

export interface ButtonProps extends JSX.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  loading?: boolean;
  fullWidth?: boolean;
  children: JSX.Element;
}

export const Button: Component<ButtonProps> = (props) => {
  const merged = mergeProps({ variant: 'primary' as const, size: 'md' as const }, props);
  const [local, others] = splitProps(merged, ['variant', 'size', 'loading', 'fullWidth', 'children', 'class', 'disabled']);

  const buttonClass = () => {
    const classes = [
      'btn',
      `btn-${local.variant}`,
      local.size !== 'md' && `btn-${local.size}`,
      local.loading && 'btn-loading',
      local.fullWidth && 'w-full',
      local.class
    ].filter(Boolean);
    
    return classes.join(' ');
  };

  const isDisabled = () => local.disabled || local.loading;

  return (
    <button
      type="button"
      class={buttonClass()}
      disabled={isDisabled()}
      {...others}
    >
      <Show when={local.loading}>
        <span class="spinner w-4 h-4 mr-2" data-testid="loading-spinner" />
      </Show>
      {local.children}
    </button>
  );
};