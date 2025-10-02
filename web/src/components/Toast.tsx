import { useEffect } from 'react';

export interface ToastProps {
  message: string;
  type?: 'success' | 'error' | 'info';
  duration?: number;
  onClose: () => void;
}

export default function Toast({ message, type = 'info', duration = 3000, onClose }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onClose();
    }, duration);

    return () => clearTimeout(timer);
  }, [duration, onClose]);

  const bgColorClass = {
    success: 'bg-green-500',
    error: 'bg-red-500',
    info: 'bg-blue-500',
  }[type];

  const icon = {
    success: '✓',
    error: '✗',
    info: 'ℹ',
  }[type];

  return (
    <div className="fixed top-4 right-4 z-50 animate-slide-in">
      <div className={`${bgColorClass} text-white px-6 py-4 rounded-lg shadow-lg flex items-center space-x-3 min-w-[320px] max-w-md`}>
        <div className="flex-shrink-0 w-6 h-6 rounded-full bg-white bg-opacity-30 flex items-center justify-center font-bold">
          {icon}
        </div>
        <div className="flex-1 text-sm font-medium">
          {message}
        </div>
        <button
          onClick={onClose}
          className="flex-shrink-0 text-white hover:text-gray-200 focus:outline-none"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  );
}
