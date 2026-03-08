import { html, useState, useCallback } from '../lib.js';

export function useToast() {
  const [toasts, setToasts] = useState([]);

  const addToast = useCallback((message, type = 'info') => {
    const id = Date.now() + Math.random();
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
    }, 4000);
  }, []);

  const removeToast = useCallback((id) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  return { toasts, addToast, removeToast };
}

export function ToastContainer({ toasts, onDismiss }) {
  return html`
    <div class="toast-container">
      ${toasts.map(t => html`
        <div key=${t.id} class="toast toast-${t.type}" onClick=${() => onDismiss(t.id)}>
          ${t.message}
        </div>
      `)}
    </div>
  `;
}
