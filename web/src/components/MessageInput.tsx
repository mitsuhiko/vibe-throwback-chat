import { createSignal, Show, For, createEffect, onCleanup } from "solid-js";
import { currentChannel, chatAPI, getters, appState } from "../store";
import {
  parseCommand,
  getCommandSuggestions,
  validateCommandArgs,
  getCommandHelp,
  isPartialCommand,
  type CommandSuggestion,
} from "../utils/commands";

export function MessageInput() {
  const [message, setMessage] = createSignal("");
  const [isSending, setIsSending] = createSignal(false);
  const [isTyping, setIsTyping] = createSignal(false);
  const [showSuggestions, setShowSuggestions] = createSignal(false);
  const [suggestions, setSuggestions] = createSignal<CommandSuggestion[]>([]);
  const [selectedSuggestion, setSelectedSuggestion] = createSignal(0);
  const [commandFeedback, setCommandFeedback] = createSignal<{
    type: "error" | "success" | "info";
    message: string;
  } | null>(null);

  let textareaRef: HTMLTextAreaElement | undefined;
  let feedbackTimeoutId: number | undefined;

  const clearFeedback = () => {
    if (feedbackTimeoutId) {
      clearTimeout(feedbackTimeoutId);
    }
    setCommandFeedback(null);
  };

  const showFeedback = (
    type: "error" | "success" | "info",
    message: string,
    duration = 5000,
  ) => {
    clearFeedback();
    setCommandFeedback({ type, message });
    feedbackTimeoutId = setTimeout(clearFeedback, duration);
  };

  const executeCommand = async (
    command: string,
    args: string[],
    rawArgs: string,
    channelId: string,
  ): Promise<void> => {
    try {
      switch (command) {
        case "join":
          await chatAPI.joinChannel(args[0]);
          showFeedback("success", `Joined channel ${args[0]}`);
          break;

        case "leave":
        case "part":
          await chatAPI.leaveChannel(channelId);
          showFeedback("success", "Left channel");
          break;

        case "nick":
          await chatAPI.changeNickname(args[0]);
          showFeedback("success", `Changed nickname to ${args[0]}`);
          break;

        case "me":
          await chatAPI.sendMeAction(channelId, rawArgs);
          break;

        case "kick":
          const username = args[0];
          const reason = args.slice(1).join(" ");
          // Find user by nickname in current channel
          const channelUsers = getters.getCurrentChannelUsers();
          const userToKick = channelUsers.find((u) => u.nickname === username);
          if (!userToKick) {
            throw new Error(`User "${username}" not found in this channel`);
          }
          await chatAPI.kickUser(
            channelId.toString(),
            userToKick.id.toString(),
            reason || undefined,
          );
          showFeedback(
            "success",
            `Kicked ${username}${reason ? ` (${reason})` : ""}`,
          );
          break;

        case "topic":
          await chatAPI.changeTopic(channelId, rawArgs);
          showFeedback("success", "Topic changed");
          break;

        case "announce":
          await chatAPI.announce(rawArgs, channelId);
          showFeedback("success", "Announcement sent");
          break;

        case "help":
          const helpCommand = args[0];
          const helpText = getCommandHelp(helpCommand);
          showFeedback("info", helpText, 10000); // Show help longer
          break;

        default:
          throw new Error(`Unknown command: /${command}`);
      }
    } catch (error: any) {
      showFeedback("error", error.message || "Command failed");
      throw error; // Re-throw to be handled by caller
    }
  };

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    const messageText = message().trim();
    const channelId = currentChannel();

    if (!messageText || isSending()) return;

    // Hide suggestions on submit
    setShowSuggestions(false);

    setIsSending(true);
    try {
      if (messageText.startsWith("/")) {
        // Parse and execute command
        const parsed = parseCommand(messageText);

        if (!parsed) {
          throw new Error("Invalid command");
        }

        if (!parsed.isValid) {
          throw new Error(parsed.error || "Invalid command");
        }

        // Validate command arguments
        const argError = validateCommandArgs(parsed.command, parsed.args);
        if (argError) {
          throw new Error(argError);
        }

        // Special handling for commands that don't require a channel
        if (parsed.command === "join" || parsed.command === "help") {
          await executeCommand(
            parsed.command,
            parsed.args,
            parsed.rawArgs,
            channelId || "",
          );
        } else if (!channelId) {
          throw new Error("This command requires you to be in a channel");
        } else {
          await executeCommand(
            parsed.command,
            parsed.args,
            parsed.rawArgs,
            channelId,
          );
        }
      } else {
        // Regular message
        if (!channelId) {
          throw new Error("Please join a channel first");
        }
        await chatAPI.sendMessage(channelId, messageText);
      }
      setMessage("");
    } catch (error: any) {
      console.error("Failed to send message/command:", error);
      showFeedback("error", error.message || "Failed to send");
    } finally {
      setIsSending(false);
      // Focus the input after sending (in finally block to ensure it happens)
      setTimeout(() => {
        if (textareaRef) {
          textareaRef.focus();
        }
      }, 0);
    }
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (showSuggestions()) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedSuggestion((prev) =>
          prev < suggestions().length - 1 ? prev + 1 : 0,
        );
        return;
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedSuggestion((prev) =>
          prev > 0 ? prev - 1 : suggestions().length - 1,
        );
        return;
      } else if (e.key === "Tab" || e.key === "Enter") {
        e.preventDefault();
        const suggestion = suggestions()[selectedSuggestion()];
        if (suggestion) {
          applySuggestion(suggestion);
        }
        return;
      } else if (e.key === "Escape") {
        e.preventDefault();
        setShowSuggestions(false);
        return;
      }
    }

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const applySuggestion = (suggestion: CommandSuggestion) => {
    setMessage(suggestion.command + " ");
    setShowSuggestions(false);
    if (textareaRef) {
      textareaRef.focus();
      // Place cursor at end
      setTimeout(() => {
        if (textareaRef) {
          textareaRef.selectionStart = textareaRef.selectionEnd =
            textareaRef.value.length;
        }
      }, 0);
    }
  };

  const handleInput = (e: InputEvent) => {
    const target = e.currentTarget as HTMLTextAreaElement;
    setMessage(target.value);

    // Auto-resize textarea
    target.style.height = "auto";
    target.style.height = Math.min(target.scrollHeight, 120) + "px";

    // Update command suggestions
    if (isPartialCommand(target.value)) {
      const newSuggestions = getCommandSuggestions(target.value);
      setSuggestions(newSuggestions);
      setShowSuggestions(newSuggestions.length > 0);
      setSelectedSuggestion(0);
    } else {
      setShowSuggestions(false);
    }

    // Clear any previous feedback when user starts typing
    if (commandFeedback()) {
      clearFeedback();
    }

    // Typing indicator (simple version)
    if (!isTyping() && target.value.length > 0) {
      setIsTyping(true);
      setTimeout(() => setIsTyping(false), 3000);
    }
  };

  const getCurrentChannelName = () => {
    const channelData = getters.getCurrentChannelData();
    return channelData ? channelData.name : "...";
  };

  const getPlaceholderText = () => {
    const channelName = getCurrentChannelName();
    return channelName !== "..."
      ? `Message ${channelName} or type / for commands`
      : "Join a channel first, or type /join channel";
  };

  // Cleanup timeout on unmount
  onCleanup(() => {
    if (feedbackTimeoutId) {
      clearTimeout(feedbackTimeoutId);
    }
  });

  return (
    <div class="border-t border-gray-700 bg-gray-800">
      <div class="p-4">
        {/* Command Feedback */}
        <Show when={commandFeedback()}>
          <div
            class={`mb-3 p-2 rounded-lg text-sm ${
              commandFeedback()!.type === "error"
                ? "bg-red-900/50 text-red-200 border border-red-800"
                : commandFeedback()!.type === "success"
                  ? "bg-green-900/50 text-green-200 border border-green-800"
                  : "bg-blue-900/50 text-blue-200 border border-blue-800"
            }`}
          >
            <div class="flex items-start justify-between">
              <div class="flex-1 whitespace-pre-wrap">
                {commandFeedback()!.message}
              </div>
              <button
                onClick={clearFeedback}
                class="ml-2 text-gray-400 hover:text-gray-200"
              >
                ×
              </button>
            </div>
          </div>
        </Show>

        <form onSubmit={handleSubmit} class="flex items-end space-x-3">
          {/* Message Input */}
          <div class="flex-1 relative">
            <div class="relative">
              <textarea
                ref={textareaRef}
                value={message()}
                onInput={handleInput}
                onKeyDown={handleKeyDown}
                placeholder={getPlaceholderText()}
                disabled={isSending()}
                rows="1"
                class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg text-gray-100 placeholder-gray-400 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-800 disabled:cursor-not-allowed transition-colors"
                style="min-height: 40px; max-height: 120px;"
              />

              {/* Character count */}
              <Show when={message().length > 500}>
                <div class="absolute bottom-1 right-2 text-xs text-gray-500">
                  <span
                    class={
                      message().length > 1000
                        ? "text-red-400"
                        : "text-yellow-400"
                    }
                  >
                    {message().length}/2000
                  </span>
                </div>
              </Show>
            </div>

            {/* Command Suggestions Dropdown */}
            <Show when={showSuggestions() && suggestions().length > 0}>
              <div class="absolute bottom-full left-0 right-0 mb-1 bg-gray-800 border border-gray-600 rounded-lg shadow-lg max-h-48 overflow-y-auto z-50">
                <For each={suggestions()}>
                  {(suggestion, index) => (
                    <button
                      type="button"
                      onClick={() => applySuggestion(suggestion)}
                      class={`w-full px-3 py-2 text-left hover:bg-gray-700 first:rounded-t-lg last:rounded-b-lg transition-colors ${
                        index() === selectedSuggestion() ? "bg-gray-700" : ""
                      }`}
                    >
                      <div class="text-blue-400 font-mono text-sm">
                        {suggestion.command}
                      </div>
                      <div class="text-gray-300 text-xs">
                        {suggestion.description}
                      </div>
                      <div class="text-gray-500 text-xs font-mono">
                        {suggestion.usage}
                      </div>
                    </button>
                  )}
                </For>
              </div>
            </Show>

            {/* Command validation feedback */}
            <Show when={message().startsWith("/") && message().length > 1}>
              {(() => {
                const parsed = parseCommand(message());
                if (parsed && !parsed.isValid) {
                  return (
                    <div class="mt-1 text-xs text-red-400">{parsed.error}</div>
                  );
                } else if (parsed && parsed.isValid) {
                  const argError = validateCommandArgs(
                    parsed.command,
                    parsed.args,
                  );
                  if (argError) {
                    return (
                      <div class="mt-1 text-xs text-red-400">{argError}</div>
                    );
                  } else {
                    return (
                      <div class="mt-1 text-xs text-green-400">
                        ✓ Valid command
                      </div>
                    );
                  }
                }
                return null;
              })()}
            </Show>
          </div>

          {/* Send Button */}
          <button
            type="submit"
            disabled={!message().trim() || isSending()}
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed transition-colors flex items-center space-x-2"
          >
            <Show
              when={!isSending()}
              fallback={
                <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
              }
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
                  d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
                />
              </svg>
            </Show>
            <span class="hidden sm:inline">
              {isSending() ? "Sending..." : "Send"}
            </span>
          </button>
        </form>

        {/* Help Text */}
        <div class="mt-2 flex items-center justify-between text-xs text-gray-500">
          <div class="flex items-center space-x-4">
            <span>Press Enter to send, Shift+Enter for new line</span>
            <Show when={showSuggestions()}>
              <span>↑↓ navigate, Tab/Enter select, Esc cancel</span>
            </Show>
          </div>
          <div class="flex items-center space-x-2">
            <button
              type="button"
              onClick={() => setMessage("/help ")}
              class="hover:text-gray-300 p-1 transition-colors"
              title="Show command help"
            >
              ?
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
