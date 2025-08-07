// Global type declarations for browser APIs and DOM types

// Standard Web API types that need to be available globally
interface ReadableStream<R = Uint8Array> {
  readonly locked: boolean;
  cancel(reason?: unknown): Promise<void>;
  getReader(): ReadableStreamDefaultReader<R>;
}

interface BufferSource {
  readonly byteLength: number;
}

interface EventTarget {
  addEventListener(type: string, listener: EventListener | null, options?: AddEventListenerOptions | boolean): void;
  dispatchEvent(event: Event): boolean;
  removeEventListener(type: string, listener: EventListener | null, options?: EventListenerOptions | boolean): void;
}

interface Request {
  readonly url: string;
  readonly method: string;
  readonly headers: Headers;
}

interface AbortSignal extends EventTarget {
  readonly aborted: boolean;
  readonly reason: unknown;
}

interface ReadableStreamDefaultReader<R = Uint8Array> {
  readonly closed: Promise<undefined>;
  cancel(reason?: unknown): Promise<void>;
  read(): Promise<ReadableStreamReadResult<R>>;
}

interface ReadableStreamReadResult<T> {
  done: boolean;
  value: T | undefined;
}


interface EventListener {
  (evt: Event): void;
}

interface AddEventListenerOptions extends EventListenerOptions {
  once?: boolean;
  passive?: boolean;
  signal?: AbortSignal;
}

interface EventListenerOptions {
  capture?: boolean;
}

// Storage interface
interface Storage {
  readonly length: number;
  clear(): void;
  getItem(key: string): string | null;
  key(index: number): string | null;
  removeItem(key: string): void;
  setItem(key: string, value: string): void;
}

// Blob interface
interface Blob {
  readonly size: number;
  readonly type: string;
  slice(start?: number, end?: number, contentType?: string): Blob;
  stream(): ReadableStream;
  text(): Promise<string>;
  arrayBuffer(): Promise<ArrayBuffer>;
}

// File interface
interface File extends Blob {
  readonly lastModified: number;
  readonly name: string;
}

// Headers interface
type HeadersInit = string[][] | Record<string, string> | Headers;
interface Headers {
  append(name: string, value: string): void;
  delete(name: string): void;
  get(name: string): string | null;
  has(name: string): boolean;
  set(name: string, value: string): void;
  forEach(callbackfn: (value: string, key: string, parent: Headers) => void, thisArg?: unknown): void;
}

// Body types
type BodyInit = Blob | BufferSource | FormData | URLSearchParams | ReadableStream<Uint8Array> | string;

// Request types
type RequestInfo = Request | string;

interface RequestInit {
  body?: BodyInit | null;
  headers?: HeadersInit;
  method?: string;
  mode?: 'cors' | 'navigate' | 'no-cors' | 'same-origin';
  credentials?: 'include' | 'omit' | 'same-origin';
  cache?: 'default' | 'force-cache' | 'no-cache' | 'no-store' | 'only-if-cached' | 'reload';
  redirect?: 'error' | 'follow' | 'manual';
  referrer?: string;
  referrerPolicy?: '' | 'no-referrer' | 'no-referrer-when-downgrade' | 'origin' | 'origin-when-cross-origin' | 'same-origin' | 'strict-origin' | 'strict-origin-when-cross-origin' | 'unsafe-url';
  integrity?: string;
  keepalive?: boolean;
  signal?: AbortSignal | null;
}

// Response interface
interface Response {
  readonly headers: Headers;
  readonly ok: boolean;
  readonly redirected: boolean;
  readonly status: number;
  readonly statusText: string;
  readonly type: 'basic' | 'cors' | 'default' | 'error' | 'opaque' | 'opaqueredirect';
  readonly url: string;
  readonly body: ReadableStream<Uint8Array> | null;
  readonly bodyUsed: boolean;
  
  arrayBuffer(): Promise<ArrayBuffer>;
  blob(): Promise<Blob>;
  clone(): Response;
  formData(): Promise<FormData>;
  json(): Promise<unknown>;
  text(): Promise<string>;
}

// FormData interface
interface FormData {
  append(name: string, value: string | Blob, fileName?: string): void;
  delete(name: string): void;
  get(name: string): FormDataEntryValue | null;
  getAll(name: string): FormDataEntryValue[];
  has(name: string): boolean;
  set(name: string, value: string | Blob, fileName?: string): void;
  forEach(callbackfn: (value: FormDataEntryValue, key: string, parent: FormData) => void, thisArg?: unknown): void;
}

type FormDataEntryValue = File | string;

// URLSearchParams interface
interface URLSearchParams {
  append(name: string, value: string): void;
  delete(name: string): void;
  get(name: string): string | null;
  getAll(name: string): string[];
  has(name: string): boolean;
  set(name: string, value: string): void;
  sort(): void;
  toString(): string;
  forEach(callbackfn: (value: string, key: string, parent: URLSearchParams) => void, thisArg?: unknown): void;
}

// Event interfaces
interface Event {
  readonly bubbles: boolean;
  readonly cancelable: boolean;
  readonly currentTarget: EventTarget | null;
  readonly defaultPrevented: boolean;
  readonly eventPhase: number;
  readonly target: EventTarget | null;
  readonly timeStamp: number;
  readonly type: string;
  preventDefault(): void;
  stopPropagation(): void;
  stopImmediatePropagation(): void;
}


interface UIEvent extends Event {
  readonly detail: number;
  readonly view: unknown;
}

interface CustomEvent<T = unknown> extends Event {
  readonly detail: T;
}

// HTML Element interfaces
interface Element {
  readonly tagName: string;
  id: string;
  className: string;
}

interface CSSStyleDeclaration {
  [property: string]: string;
}

interface HTMLElement extends Element {
  click(): void;
  focus(): void;
  blur(): void;
  style: CSSStyleDeclaration;
}



// Document interface
interface Document {
  createElement(tagName: string): HTMLElement;
  getElementById(elementId: string): HTMLElement | null;
  body: HTMLElement;
}

// Window interface extensions
declare global {
  interface Window {
    localStorage: Storage;
    sessionStorage: Storage;
    fetch: (input: RequestInfo, init?: RequestInit) => Promise<Response>;
    setTimeout: (handler: TimerHandler, timeout?: number, ...arguments: unknown[]) => number;
    clearTimeout: (id?: number) => void;
    setInterval: (handler: TimerHandler, timeout?: number, ...arguments: unknown[]) => number;
    clearInterval: (id?: number) => void;
    FormData: new() => FormData;
    Headers: new(init?: HeadersInit) => Headers;
    URLSearchParams: new(init?: string | string[][] | Record<string, string>) => URLSearchParams;
    URL: {
      createObjectURL(obj: Blob): string;
      revokeObjectURL(url: string): void;
    };
    CustomEvent: new<T = unknown>(type: string, eventInitDict?: CustomEventInit<T>) => CustomEvent<T>;
    dispatchEvent(event: Event): boolean;
  }

  // Global constructors
  const FormData: new() => FormData;
  const Headers: new(init?: HeadersInit) => Headers;
  const URLSearchParams: new(init?: string | string[][] | Record<string, string>) => URLSearchParams;
  const fetch: (input: RequestInfo, init?: RequestInit) => Promise<Response>;
  const setTimeout: (handler: TimerHandler, timeout?: number, ...arguments: unknown[]) => number;
  const clearTimeout: (id?: number) => void;
  const setInterval: (handler: TimerHandler, timeout?: number, ...arguments: unknown[]) => number;
  const clearInterval: (id?: number) => void;
  
  // DOM APIs
  const KeyboardEvent: {
    new(type: string, eventInitDict?: KeyboardEventInit): KeyboardEvent;
    readonly prototype: KeyboardEvent;
  };

  const HTMLFormElement: {
    new(): HTMLFormElement;
    readonly prototype: HTMLFormElement;  
  };

  const HTMLInputElement: {
    new(): HTMLInputElement;
    readonly prototype: HTMLInputElement;
  };

  const File: {
    new(fileBits: BlobPart[], fileName: string, options?: FilePropertyBag): File;
    readonly prototype: File;
  };

  // Additional types
  type TimerHandler = string | (() => void);

  // DOM Event types
  interface KeyboardEvent extends UIEvent {
    readonly altKey: boolean;
    readonly code: string;
    readonly ctrlKey: boolean;
    readonly isComposing: boolean;
    readonly key: string;
    readonly location: number;
    readonly metaKey: boolean;
    readonly repeat: boolean;
    readonly shiftKey: boolean;
  }

  // Form element types
  interface HTMLFormElement extends HTMLElement {
    readonly length: number;
    submit(): void;
    reset(): void;
    requestSubmit(): void;
  }

  interface HTMLInputElement extends HTMLElement {
    value: string;
    type: string;
    checked: boolean;
    disabled: boolean;
    readonly: boolean;
  }

  interface CustomEventInit<T = unknown> extends EventInit {
    detail?: T;
  }

  interface EventInit {
    bubbles?: boolean;
    cancelable?: boolean;
    composed?: boolean;
  }

  interface KeyboardEventInit extends EventInit {
    key?: string;
    code?: string;
    location?: number;
    ctrlKey?: boolean;
    shiftKey?: boolean;
    altKey?: boolean;
    metaKey?: boolean;
    repeat?: boolean;
    isComposing?: boolean;
  }

  interface FilePropertyBag {
    type?: string;
    lastModified?: number;
  }

  type BlobPart = BufferSource | Blob | string;

  // Document and console globals
  const document: Document;
  const console: Console;

  interface Console {
    log(...data: unknown[]): void;
    error(...data: unknown[]): void;
    warn(...data: unknown[]): void;
    info(...data: unknown[]): void;
  }
}

export {};