import { html, useState, useEffect, useCallback, useRef } from '../lib.js';

export function useToast() {
  const [toasts, setToasts] = useState([]);
  const timersRef = useRef({});

  useEffect(() => {
    return () => {
      for (const id in timersRef.current) clearTimeout(timersRef.current[id]);
    };
  }, []);

  const addToast = useCallback((message, type = 'info') => {
    const id = Date.now() + Math.random();
    setToasts(prev => [...prev, { id, message, type }]);
    timersRef.current[id] = setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
      delete timersRef.current[id];
    }, 4000);
  }, []);

  const removeToast = useCallback((id) => {
    clearTimeout(timersRef.current[id]);
    delete timersRef.current[id];
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
