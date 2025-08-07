import type { Component, JSX } from 'solid-js';
import { createSignal, createUniqueId, mergeProps, Show, splitProps } from 'solid-js';

export interface InputProps extends Omit<JSX.InputHTMLAttributes<HTMLInputElement>, 'onChange'> {
  label?: string;
  error?: string;
  helperText?: string;
  showPasswordToggle?: boolean;
  onChange?: (value: string) => void;
}

export const Input: Component<InputProps> = (props) => {
  const merged = mergeProps({ type: 'text' as const }, props);
  const [local, others] = splitProps(merged, [
    'label', 'error', 'helperText', 'showPasswordToggle', 'onChange', 'type', 'class', 'id'
  ]);

  const [showPassword, setShowPassword] = createSignal(false);
  const inputId = local.id || createUniqueId();
  const helperId = createUniqueId();

  const inputType = () => {
    if (local.type === 'password' && local.showPasswordToggle) {
      return showPassword() ? 'text' : 'password';
    }
    return local.type;
  };

  const inputClass = () => {
    const classes = [
      'form-input',
      local.error && 'form-input--error',
      local.showPasswordToggle && local.type === 'password' && 'pr-10',
      local.class
    ].filter(Boolean);
    
    return classes.join(' ');
  };

  const handleInput = (e: Event) => {
    const target = e.target as HTMLInputElement;
    local.onChange?.(target.value);
  };

  const togglePassword = () => {
    setShowPassword(!showPassword());
  };

  const getAriaDescribedBy = () => {
    const describedBy = [];
    if (local.helperText) describedBy.push(helperId);
    if (local.error) describedBy.push(`${inputId}-error`);
    return describedBy.length > 0 ? describedBy.join(' ') : undefined;
  };

  return (
    <div class="space-y-1">
      <Show when={local.label}>
        <label for={inputId} class="form-label block text-ds-14 font-medium text-ds-text">
          {local.label}
          <Show when={others.required}>
            <span class="text-ds-danger-bold ml-1">*</span>
          </Show>
        </label>
      </Show>
      
      <div class="relative">
        <input
          id={inputId}
          type={inputType()}
          class={inputClass()}
          aria-describedby={getAriaDescribedBy()}
          onInput={handleInput}
          {...others}
        />
        
        <Show when={local.showPasswordToggle && local.type === 'password'}>
          <button
            type="button"
            class="absolute inset-y-0 right-0 pr-3 flex items-center"
            onClick={togglePassword}
          >
            <Show when={showPassword()} fallback={
              <svg class="h-4 w-4 text-ds-icon-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
            }>
              <svg class="h-4 w-4 text-ds-icon-subtle" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21" />
              </svg>
            </Show>
          </button>
        </Show>
      </div>
      
      <Show when={local.error}>
        <p id={`${inputId}-error`} class="text-ds-12 text-ds-danger-bold">
          {local.error}
        </p>
      </Show>
      
      <Show when={local.helperText && !local.error}>
        <p id={helperId} class="text-ds-12 text-ds-text-subtle">
          {local.helperText}
        </p>
      </Show>
    </div>
  );
};