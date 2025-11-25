import { useEffect } from "react";

export type ToastType = "info" | "success" | "error";

export interface ToastMessage {
  id: string;
  message: string;
  type: ToastType;
  duration?: number;
  link?: string;
  linkText?: string;
}

interface ToastProps {
  messages: ToastMessage[];
  onDismiss: (id: string) => void;
}

export const Toast = ({ messages, onDismiss }: ToastProps) => {
  useEffect(() => {
    messages.forEach((msg) => {
      const duration = msg.duration || 5000;
      const timer = setTimeout(() => {
        onDismiss(msg.id);
      }, duration);

      return () => clearTimeout(timer);
    });
  }, [messages, onDismiss]);

  if (messages.length === 0) return null;

  const getAlertClass = (type: ToastType) => {
    switch (type) {
      case "success":
        return "alert-success";
      case "error":
        return "alert-error";
      case "info":
      default:
        return "alert-info";
    }
  };

  return (
    <div className="toast toast-top toast-end z-50">
      {messages.map((msg) => (
        <div key={msg.id} className={`alert ${getAlertClass(msg.type)} shadow-lg`}>
          <div className="flex items-center gap-2">
            <span>{msg.message}</span>
            {msg.link && (
              <a
                href={msg.link}
                className="btn btn-xs btn-ghost underline"
                onClick={() => onDismiss(msg.id)}
              >
                {msg.linkText || "View Details"}
              </a>
            )}
            <button
              onClick={() => onDismiss(msg.id)}
              className="btn btn-ghost btn-xs btn-circle"
            >
              ✕
            </button>
          </div>
        </div>
      ))}
    </div>
  );
};
