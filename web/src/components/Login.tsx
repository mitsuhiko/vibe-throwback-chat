import { createSignal, createEffect } from "solid-js";
import { chatAPI, connectionState } from "../store";

interface LoginProps {
  onLoginSuccess?: () => void;
}

export function Login(props: LoginProps) {
  const [nickname, setNickname] = createSignal("");
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  // Auto-connect when component mounts (only if not already connected/connecting)
  createEffect(() => {
    if (connectionState() === "disconnected") {
      chatAPI.connect();
    }
  });

  // Clear error when user starts typing
  createEffect(() => {
    if (nickname()) {
      setError(null);
    }
  });

  const handleSubmit = async (e: Event) => {
    e.preventDefault();

    const nick = nickname().trim();
    if (!nick) {
      setError("Please enter a nickname");
      return;
    }

    if (nick.length < 2) {
      setError("Nickname must be at least 2 characters");
      return;
    }

    if (nick.length > 20) {
      setError("Nickname must be 20 characters or less");
      return;
    }

    // Basic nickname validation (alphanumeric + some special chars)
    if (!/^[a-zA-Z0-9_\-\[\]{}^`|\\]+$/.test(nick)) {
      setError("Nickname contains invalid characters");
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      await chatAPI.login(nick);
      props.onLoginSuccess?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setIsLoading(false);
    }
  };

  const getConnectionStatusColor = () => {
    switch (connectionState()) {
      case "connected":
        return "text-green-400";
      case "connecting":
      case "reconnecting":
        return "text-yellow-400";
      case "error":
        return "text-red-400";
      default:
        return "text-gray-400";
    }
  };

  const getConnectionStatusText = () => {
    switch (connectionState()) {
      case "connected":
        return "Connected";
      case "connecting":
        return "Connecting...";
      case "reconnecting":
        return "Reconnecting...";
      case "error":
        return "Connection Error";
      default:
        return "Disconnected";
    }
  };

  return (
    <div class="min-h-screen bg-gray-900 flex items-center justify-center p-8">
      <div class="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-8 max-w-md w-full">
        <h1 class="text-3xl font-bold text-gray-100 mb-6 text-center">
          ThrowBackChat
        </h1>

        <div class="space-y-6">
          {/* Connection Status */}
          <div class="text-center">
            <div class={`text-sm font-medium ${getConnectionStatusColor()}`}>
              {getConnectionStatusText()}
            </div>
          </div>

          {/* Login Form */}
          <form onSubmit={handleSubmit} class="space-y-4">
            <div>
              <label
                for="nickname"
                class="block text-sm font-medium text-gray-300 mb-2"
              >
                Choose your nickname
              </label>
              <input
                id="nickname"
                type="text"
                value={nickname()}
                onInput={(e) => setNickname(e.currentTarget.value)}
                placeholder="Enter nickname..."
                class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md shadow-sm text-gray-100 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-400 focus:border-blue-400"
                disabled={isLoading() || connectionState() !== "connected"}
                maxLength={20}
                required
              />
              <div class="mt-1 text-xs text-gray-400">
                2-20 characters, letters, numbers, and _-[]{}^`|\\ allowed
              </div>
            </div>

            {error() && (
              <div class="text-red-400 text-sm bg-red-900 border border-red-700 rounded-md p-3">
                {error()}
              </div>
            )}

            <button
              type="submit"
              disabled={
                isLoading() ||
                !nickname().trim() ||
                connectionState() !== "connected"
              }
              class="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-400 focus:ring-offset-2 focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isLoading() ? (
                <div class="flex items-center justify-center">
                  <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                  Logging in...
                </div>
              ) : (
                "Join Chat"
              )}
            </button>
          </form>

          {/* Instructions */}
          <div class="text-center text-sm text-gray-400 space-y-2">
            <p>Welcome to ThrowBackChat!</p>
            <p>Enter a nickname to join the conversation.</p>
            <p>
              {connectionState() !== "connected" &&
                "Waiting for server connection..."}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
