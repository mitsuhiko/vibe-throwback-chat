import { For, Show } from "solid-js";
import {
  getTempNotifications,
  removeTempNotification,
  currentChannel,
} from "../store";

export function TempNotifications() {
  const notifications = getTempNotifications();

  // Filter notifications for current channel (or global ones)
  const relevantNotifications = () => {
    const channelId = currentChannel();
    return notifications.filter(
      (n) => !n.channelId || n.channelId === channelId,
    );
  };

  return (
    <Show when={relevantNotifications().length > 0}>
      <div class="fixed top-4 right-4 z-50 space-y-2 max-w-sm">
        <For each={relevantNotifications()}>
          {(notification) => (
            <div
              class={`p-3 rounded-lg shadow-lg border backdrop-blur-sm animate-in slide-in-from-right-4 fade-in-25 duration-300 ${
                notification.type === "error"
                  ? "bg-red-900/90 text-red-200 border-red-800"
                  : notification.type === "success"
                    ? "bg-green-900/90 text-green-200 border-green-800"
                    : "bg-blue-900/90 text-blue-200 border-blue-800"
              }`}
            >
              <div class="flex items-start justify-between">
                <div class="flex-1 text-sm whitespace-pre-wrap">
                  {notification.message}
                </div>
                <button
                  onClick={() => removeTempNotification(notification.id)}
                  class="ml-2 text-current opacity-70 hover:opacity-100 transition-opacity"
                  aria-label="Dismiss notification"
                >
                  ×
                </button>
              </div>
            </div>
          )}
        </For>
      </div>
    </Show>
  );
}
