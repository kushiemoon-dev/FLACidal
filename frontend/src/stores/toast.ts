import { writable } from 'svelte/store';
import type { MessageKey, PluralKey, UiMsg } from '../lib/i18n';
import type { Vars } from '../lib/i18n/translate';

type ToastType = 'success' | 'error' | 'info';

interface Toast {
  id: number;
  /** Raw text, or a key translated at render time so it follows language switches. */
  message: NonNullable<UiMsg>;
  type: ToastType;
}

function createToastStore() {
  const { subscribe, update } = writable<Toast[]>([]);
  let nextId = 0;

  function show(message: NonNullable<UiMsg>, type: ToastType = 'success', duration = 3000) {
    const id = nextId++;
    update(toasts => [...toasts, { id, message, type }]);
    setTimeout(() => {
      update(toasts => toasts.filter(t => t.id !== id));
    }, duration);
  }

  return {
    subscribe,
    /** Raw text (already final, e.g. a Core error) or a `{ key, vars }` message. */
    show,
    /** Translatable message; the text is resolved when rendered. */
    showKey(key: MessageKey | PluralKey, vars?: Vars, type: ToastType = 'success', duration = 3000) {
      show({ key, vars }, type, duration);
    },
    /** Raw text when present (Core/network error), otherwise the translatable fallback. */
    showMsg(message: UiMsg, type: ToastType = 'success', duration = 3000) {
      if (message) show(message, type, duration);
    }
  };
}

export const toastStore = createToastStore();
