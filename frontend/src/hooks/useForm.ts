import { createSignal } from 'solid-js';
import { FormValidator, createFieldState } from '../utils/validation';

export interface FormFieldState {
  value: string;
  touched: boolean;
  errors: string[];
  isValid: boolean;
}

export type FormState<T extends Record<string, string>> = {
  [K in keyof T]: FormFieldState;
}

export interface FormActions<T extends Record<string, string>> {
  setValue: (field: keyof T, value: string) => void;
  setTouched: (field: keyof T, touched: boolean) => void;
  setError: (field: keyof T, errors: string[]) => void;
  reset: (initialValues?: Partial<T>) => void;
  validate: () => boolean;
  validateField: (field: keyof T) => boolean;
  getValues: () => T;
  getErrors: () => Record<keyof T, string[]>;
}

export interface UseFormOptions<T extends Record<string, string>> {
  initialValues?: Partial<T>;
  validator?: FormValidator;
  validateOnChange?: boolean;
  validateOnBlur?: boolean;
}

export function useForm<T extends Record<string, string>>(
  fields: (keyof T)[],
  options: UseFormOptions<T> = {}
): [() => FormState<T>, FormActions<T>] {
  const {
    initialValues = {},
    validator,
    validateOnChange = true,
    validateOnBlur = true,
  } = options;

  // Initialize form state
  const initialFormState = fields.reduce((acc, field) => {
    const fieldValue = initialValues ? (initialValues as Record<string, string>)[field] : '';
    acc[field] = createFieldState(fieldValue || '');
    return acc;
  }, {} as FormState<T>);

  const [formState, setFormState] = createSignal<FormState<T>>(initialFormState);


  // Actions
  const actions: FormActions<T> = {
    setValue: (field: keyof T, value: string) => {
      setFormState(prev => {
        const currentField = prev[field];
        const allValues = Object.keys(prev).reduce((acc, key) => {
          acc[key] = key === field ? value : prev[key as keyof T].value;
          return acc;
        }, {} as Record<string, string>);

        let updatedField = { ...currentField, value, touched: true };

        // Validate on change if enabled
        if (validateOnChange && validator) {
          const result = validator.validateField(field as string, value, allValues);
          updatedField = {
            ...updatedField,
            errors: result.errors,
            isValid: result.isValid,
          };
        }

        return {
          ...prev,
          [field]: updatedField,
        };
      });
    },

    setTouched: (field: keyof T, touched: boolean) => {
      setFormState(prev => ({
        ...prev,
        [field]: {
          ...prev[field],
          touched,
        },
      }));

      // Validate on blur if enabled and field is being blurred
      if (!touched && validateOnBlur && validator) {
        actions.validateField(field);
      }
    },

    setError: (field: keyof T, errors: string[]) => {
      setFormState(prev => ({
        ...prev,
        [field]: {
          ...prev[field],
          errors,
          isValid: errors.length === 0,
        },
      }));
    },

    reset: (newInitialValues?: Partial<T>) => {
      const resetValues = { ...initialValues, ...newInitialValues };
      const resetState = fields.reduce((acc, field) => {
        const fieldValue = resetValues ? (resetValues as Record<string, string>)[field] : '';
        acc[field] = createFieldState(fieldValue || '');
        return acc;
      }, {} as FormState<T>);
      setFormState(() => resetState);
    },

    validate: () => {
      if (!validator) return true;

      const currentValues = actions.getValues();
      const results = validator.validateForm(currentValues);

      setFormState(prev => {
        const updated = { ...prev };
        for (const [field, result] of Object.entries(results)) {
          if (updated[field as keyof T]) {
            updated[field as keyof T] = {
              ...updated[field as keyof T],
              errors: result.errors,
              isValid: result.isValid,
              touched: true,
            };
          }
        }
        return updated;
      });

      return validator.isFormValid(currentValues);
    },

    validateField: (field: keyof T) => {
      if (!validator) return true;

      const currentValues = actions.getValues();
      const result = validator.validateField(field as string, currentValues[field], currentValues);

      setFormState(prev => ({
        ...prev,
        [field]: {
          ...prev[field],
          errors: result.errors,
          isValid: result.isValid,
          touched: true,
        },
      }));

      return result.isValid;
    },

    getValues: () => {
      const state = formState();
      return Object.keys(state).reduce((acc, key) => {
        acc[key as keyof T] = state[key as keyof T].value as T[keyof T];
        return acc;
      }, {} as T);
    },

    getErrors: () => {
      const state = formState();
      return Object.keys(state).reduce((acc, key) => {
        const field = state[key as keyof T];
        if (field.errors.length > 0) {
          acc[key as keyof T] = field.errors;
        }
        return acc;
      }, {} as Record<keyof T, string[]>);
    },
  };

  // Return the form state and actions
  return [formState, actions];
}

// Helper hook for simple forms
export function useSimpleForm<T extends Record<string, string>>(
  initialValues: T,
  validator?: FormValidator
) {
  const fields = Object.keys(initialValues) as (keyof T)[];
  return useForm<T>(fields, { initialValues, validator });
}