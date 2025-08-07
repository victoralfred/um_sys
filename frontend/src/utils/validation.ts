// Form validation utilities

export type ValidationRule<T = any> = {
  validate: (value: T, formData?: Record<string, any>) => boolean;
  message: string;
};

export type ValidationResult = {
  isValid: boolean;
  errors: string[];
};

export type FieldValidation = {
  [fieldName: string]: ValidationRule[];
};

// Common validation rules
export const validationRules = {
  required: (message = 'This field is required'): ValidationRule<any> => ({
    validate: (value) => {
      if (typeof value === 'string') return value.trim().length > 0;
      return value !== null && value !== undefined && value !== '';
    },
    message,
  }),

  email: (message = 'Please enter a valid email address'): ValidationRule<string> => ({
    validate: (value) => {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      return emailRegex.test(value.trim());
    },
    message,
  }),

  minLength: (min: number, message?: string): ValidationRule<string> => ({
    validate: (value) => value.length >= min,
    message: message || `Must be at least ${min} characters long`,
  }),

  maxLength: (max: number, message?: string): ValidationRule<string> => ({
    validate: (value) => value.length <= max,
    message: message || `Must be no more than ${max} characters long`,
  }),

  password: (message = 'Password must be at least 8 characters with uppercase, lowercase, number, and special character'): ValidationRule<string> => ({
    validate: (value) => {
      const passwordRegex = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$/;
      return passwordRegex.test(value);
    },
    message,
  }),

  confirmPassword: (message = 'Passwords do not match'): ValidationRule<string> => ({
    validate: (value, formData) => {
      return value === formData?.password;
    },
    message,
  }),

  username: (message = 'Username must be 3-30 characters, letters, numbers, underscores, and hyphens only'): ValidationRule<string> => ({
    validate: (value) => {
      const usernameRegex = /^[a-zA-Z0-9_-]{3,30}$/;
      return usernameRegex.test(value);
    },
    message,
  }),

  phoneNumber: (message = 'Please enter a valid phone number'): ValidationRule<string> => ({
    validate: (value) => {
      if (!value.trim()) return true; // Optional field
      const phoneRegex = /^\+?[\d\s\-\(\)]{10,}$/;
      return phoneRegex.test(value.replace(/\s/g, ''));
    },
    message,
  }),

  name: (message = 'Name must contain only letters and spaces'): ValidationRule<string> => ({
    validate: (value) => {
      if (!value.trim()) return true; // Optional field
      const nameRegex = /^[a-zA-Z\s]{2,50}$/;
      return nameRegex.test(value.trim());
    },
    message,
  }),
};

// Validation engine
export class FormValidator {
  private rules: FieldValidation;

  constructor(rules: FieldValidation) {
    this.rules = rules;
  }

  validateField(fieldName: string, value: any, formData?: Record<string, any>): ValidationResult {
    const fieldRules = this.rules[fieldName] || [];
    const errors: string[] = [];

    for (const rule of fieldRules) {
      if (!rule.validate(value, formData)) {
        errors.push(rule.message);
      }
    }

    return {
      isValid: errors.length === 0,
      errors,
    };
  }

  validateForm(formData: Record<string, any>): Record<string, ValidationResult> {
    const results: Record<string, ValidationResult> = {};

    for (const fieldName of Object.keys(this.rules)) {
      results[fieldName] = this.validateField(fieldName, formData[fieldName], formData);
    }

    return results;
  }

  isFormValid(formData: Record<string, any>): boolean {
    const results = this.validateForm(formData);
    return Object.values(results).every(result => result.isValid);
  }

  getFormErrors(formData: Record<string, any>): Record<string, string[]> {
    const results = this.validateForm(formData);
    const errors: Record<string, string[]> = {};

    for (const [fieldName, result] of Object.entries(results)) {
      if (!result.isValid) {
        errors[fieldName] = result.errors;
      }
    }

    return errors;
  }
}

// Predefined form validators
export const authValidators = {
  login: new FormValidator({
    email: [validationRules.required(), validationRules.email()],
    password: [validationRules.required()],
  }),

  register: new FormValidator({
    email: [validationRules.required(), validationRules.email()],
    username: [validationRules.required(), validationRules.username()],
    password: [validationRules.required(), validationRules.password()],
    confirmPassword: [validationRules.required(), validationRules.confirmPassword()],
    firstName: [validationRules.name()],
    lastName: [validationRules.name()],
  }),

  profile: new FormValidator({
    email: [validationRules.required(), validationRules.email()],
    username: [validationRules.required(), validationRules.username()],
    firstName: [validationRules.name()],
    lastName: [validationRules.name()],
    phoneNumber: [validationRules.phoneNumber()],
  }),

  changePassword: new FormValidator({
    currentPassword: [validationRules.required()],
    newPassword: [validationRules.required(), validationRules.password()],
    confirmPassword: [validationRules.required(), validationRules.confirmPassword()],
  }),
};

// Form field state helpers
export const createFieldState = (initialValue: string = '') => {
  return {
    value: initialValue,
    touched: false,
    errors: [] as string[],
    isValid: true,
  };
};

export const updateFieldState = (
  field: ReturnType<typeof createFieldState>,
  value: string,
  validator?: FormValidator,
  fieldName?: string,
  formData?: Record<string, any>
) => {
  const newField = {
    ...field,
    value,
    touched: true,
  };

  if (validator && fieldName) {
    const result = validator.validateField(fieldName, value, formData);
    newField.errors = result.errors;
    newField.isValid = result.isValid;
  }

  return newField;
};