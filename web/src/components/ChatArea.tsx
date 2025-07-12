import { For, Show, createEffect, onMount, createSignal } from "solid-js";
import { marked } from "marked";
import { getters, chatAPI } from "../store";
import type { Message } from "../types";

export function ChatArea() {
  let messagesContainer: HTMLDivElement;
  const [isLoadingHistory, setIsLoadingHistory] = createSignal(false);
  const [hasMoreHistory, setHasMoreHistory] = createSignal(true);
  const [shouldAutoScroll, setShouldAutoScroll] = createSignal(true);

  // Configure marked for better security and performance
  marked.setOptions({
    breaks: true,
    gfm: true,
  });

  // Auto-scroll to bottom when new messages arrive, but only if already at bottom
  createEffect(() => {
    const messages = getters.getCurrentChannelMessages();
    if (messages.length > 0 && messagesContainer! && shouldAutoScroll()) {
      requestAnimationFrame(() => {
        messagesContainer!.scrollTop = messagesContainer!.scrollHeight;
      });
    }
  });

  // Check if user is at bottom to determine auto-scroll behavior
  const handleScroll = () => {
    if (!messagesContainer!) return;

    const { scrollTop, scrollHeight, clientHeight } = messagesContainer!;
    const atBottom = scrollHeight - scrollTop - clientHeight < 10;
    setShouldAutoScroll(atBottom);

    // Load more history when scrolling to top
    if (scrollTop < 100 && hasMoreHistory() && !isLoadingHistory()) {
      loadMoreHistory();
    }
  };

  const loadMoreHistory = async () => {
    const currentChannel = getters.getCurrentChannelData();
    const messages = getters.getCurrentChannelMessages();

    if (!currentChannel || messages.length === 0) return;

    setIsLoadingHistory(true);

    try {
      // Get the oldest message ID for pagination
      const oldestMessage = messages[0];
      const beforeId =
        oldestMessage && oldestMessage.id
          ? parseInt(oldestMessage.id.replace(/^\w+_/, "").split("_")[0])
          : undefined;

      const { messages: newMessages, hasMore } = await chatAPI.getHistory(
        currentChannel.id,
        beforeId,
        50,
      );

      if (newMessages.length > 0) {
        // TODO: Implement proper message history prepending in the store
        // For now, just log that we received the messages
        console.log(`Loaded ${newMessages.length} historical messages`);
      }

      setHasMoreHistory(hasMore);
    } catch (error) {
      console.error("Failed to load message history:", error);
    } finally {
      setIsLoadingHistory(false);
    }
  };

  // Scroll to bottom when component mounts
  onMount(() => {
    if (messagesContainer!) {
      messagesContainer!.scrollTop = messagesContainer!.scrollHeight;
    }
  });

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffDays = Math.floor(
      (now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24),
    );

    if (diffDays === 0) {
      // Today - show time only
      return date.toLocaleTimeString("en-US", {
        hour12: false,
        hour: "2-digit",
        minute: "2-digit",
      });
    } else if (diffDays === 1) {
      // Yesterday
      return `Yesterday ${date.toLocaleTimeString("en-US", {
        hour12: false,
        hour: "2-digit",
        minute: "2-digit",
      })}`;
    } else if (diffDays < 7) {
      // This week - show day and time
      return date.toLocaleDateString("en-US", {
        weekday: "short",
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
      });
    } else {
      // Older - show full date
      return date.toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
      });
    }
  };

  const formatMessage = (message: string) => {
    try {
      // Use marked for full markdown support
      const html = marked.parse(message, { async: false }) as string;
      // Basic sanitization - remove script tags and on* attributes
      return html
        .replace(/<script[^>]*>.*?<\/script>/gi, "")
        .replace(/on\w+\s*=\s*[\"'][^\"']*[\"']/gi, "")
        .replace(/javascript:/gi, "");
    } catch (error) {
      console.error("Failed to parse markdown:", error);
      // Fallback to simple text
      return message;
    }
  };

  const getMessageTypeStyles = (message: Message) => {
    if (message.event !== "message") {
      // Event messages
      return "text-gray-400 italic opacity-75";
    } else if (message.is_passive) {
      // /me action messages
      return "text-purple-300 italic";
    } else {
      // Regular messages
      return "text-gray-100";
    }
  };

  const getAvatarStyles = (message: Message) => {
    if (message.event !== "message") {
      return "w-8 h-8 bg-gray-600 rounded-full flex items-center justify-center";
    } else if (message.is_passive) {
      return "w-8 h-8 bg-purple-600 rounded-full flex items-center justify-center";
    } else {
      return "w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center";
    }
  };

  const getAvatarContent = (message: Message) => {
    if (message.event !== "message") {
      return <span class="text-xs text-gray-400">!</span>;
    } else {
      return (
        <span class="text-sm font-medium text-white">
          {message.nickname.charAt(0).toUpperCase()}
        </span>
      );
    }
  };

  return (
    <div class="h-full flex flex-col bg-gray-900">
      {/* Messages Container */}
      <div
        ref={messagesContainer!}
        class="flex-1 overflow-y-auto p-4 space-y-2"
        onScroll={handleScroll}
      >
        {/* Loading indicator for history */}
        <Show when={isLoadingHistory()}>
          <div class="flex justify-center py-4">
            <div class="text-gray-500 text-sm flex items-center space-x-2">
              <div class="animate-spin w-4 h-4 border-2 border-gray-500 border-t-transparent rounded-full"></div>
              <span>Loading message history...</span>
            </div>
          </div>
        </Show>

        <Show
          when={getters.getCurrentChannelMessages().length > 0}
          fallback={
            <div class="flex items-center justify-center h-full">
              <div class="text-center text-gray-500">
                <div class="text-6xl mb-4">#</div>
                <h3 class="text-lg font-semibold mb-2">
                  Welcome to {getters.getCurrentChannelData()?.name}
                </h3>
                <p class="text-sm">
                  This is the beginning of the{" "}
                  {getters.getCurrentChannelData()?.name} channel.
                </p>
              </div>
            </div>
          }
        >
          <For each={getters.getCurrentChannelMessages()}>
            {(message) => (
              <div
                class={`
                group flex items-start space-x-3 hover:bg-gray-800 hover:bg-opacity-30 px-2 py-1.5 rounded transition-colors
                ${message.event !== "message" ? "opacity-80" : ""}
              `}
              >
                {/* Avatar/Icon */}
                <div class="flex-shrink-0 mt-0.5">
                  <div class={getAvatarStyles(message)}>
                    {getAvatarContent(message)}
                  </div>
                </div>

                {/* Message Content */}
                <div class="flex-1 min-w-0">
                  <div class="flex items-baseline space-x-2 mb-1">
                    <span
                      class={`
                      font-medium text-sm
                      ${
                        message.event !== "message"
                          ? "text-gray-400"
                          : message.is_passive
                            ? "text-purple-300"
                            : "text-gray-200"
                      }
                    `}
                    >
                      {message.nickname}
                    </span>
                    <span class="text-xs text-gray-500 font-mono">
                      {formatTime(message.sent_at)}
                    </span>
                  </div>

                  <div
                    class={`text-sm break-words ${getMessageTypeStyles(message)}`}
                  >
                    <Show
                      when={message.event === "message"}
                      fallback={
                        <div class="flex items-center space-x-2">
                          <span class="text-gray-500">•</span>
                          <span>{message.message}</span>
                        </div>
                      }
                    >
                      <div
                        innerHTML={formatMessage(message.message)}
                        class="prose prose-sm prose-invert max-w-none"
                      />
                    </Show>
                  </div>
                </div>

                {/* Message Actions (appear on hover) */}
                <div class="opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    class="text-gray-500 hover:text-gray-300 text-xs p-1 rounded hover:bg-gray-700"
                    title="Message options"
                  >
                    ⋯
                  </button>
                </div>
              </div>
            )}
          </For>

          {/* Scroll to bottom indicator */}
          <Show when={!shouldAutoScroll()}>
            <div class="fixed bottom-20 right-6 z-10">
              <button
                onClick={() => {
                  setShouldAutoScroll(true);
                  if (messagesContainer!) {
                    messagesContainer!.scrollTop =
                      messagesContainer!.scrollHeight;
                  }
                }}
                class="bg-blue-600 hover:bg-blue-700 text-white p-2 rounded-full shadow-lg transition-colors"
                title="Scroll to bottom"
              >
                <svg
                  class="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M19 14l-7 7m0 0l-7-7m7 7V3"
                  />
                </svg>
              </button>
            </div>
          </Show>
        </Show>
      </div>
    </div>
  );
}
