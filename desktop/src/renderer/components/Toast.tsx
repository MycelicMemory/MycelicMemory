import { Toaster, toast } from 'react-hot-toast';

export function ToastProvider() {
  return (
    <Toaster
      position="bottom-right"
      toastOptions={{
        duration: 3000,
        style: {
          background: '#1e293b',
          color: '#e2e8f0',
          border: '1px solid #334155',
          borderRadius: '0.75rem',
          fontSize: '0.875rem',
        },
        success: {
          iconTheme: { primary: '#22c55e', secondary: '#1e293b' },
        },
        error: {
          duration: 5000,
          iconTheme: { primary: '#ef4444', secondary: '#1e293b' },
        },
      }}
    />
  );
}

export { toast };
