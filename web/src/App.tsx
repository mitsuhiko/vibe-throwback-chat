import { createSignal, createEffect, Show } from "solid-js";
import { currentUser, connectionState, chatAPI } from "./store";
import { wsClient } from "./websocket";
import { Login } from "./components/Login";
import { ChatLayout } from "./components/ChatLayout";

function App() {
  const [health, setHealth] = createSignal<any>(null);
  const [healthLoading, setHealthLoading] = createSignal(true);
  const [showDebug, setShowDebug] = createSignal(false);
  const [sessionRestoreAttempted, setSessionRestoreAttempted] =
    createSignal(false);

  // Check server health on mount
  createEffect(async () => {
    try {
      const response = await fetch("/api/health");
      const data = await response.json();
      setHealth(data);
    } catch (error) {
      console.error("Failed to fetch health:", error);
    } finally {
      setHealthLoading(false);
    }
  });

  // Attempt session restoration on app load
  createEffect(() => {
    if (!sessionRestoreAttempted() && !healthLoading()) {
      const storedSessionId = wsClient.getSessionId();
      if (storedSessionId) {
        console.log("Found stored session, attempting to connect...");
        chatAPI.connect();
      } else {
        console.log("No stored session found");
      }
      setSessionRestoreAttempted(true);
    }
  });

  const DebugOverlay = () => (
    <Show when={showDebug()}>
      <div class="absolute top-4 right-4 bg-black bg-opacity-75 text-white p-4 rounded-lg text-xs z-50 max-w-xs">
        <h3 class="font-semibold mb-2">Debug Information</h3>
        <div class="space-y-1">
          <div>User ID: {currentUser()?.id}</div>
          <div>Nickname: {currentUser()?.nickname}</div>
          <div>Connection: {connectionState()}</div>
          <div>
            Health: {healthLoading() ? "Loading..." : health() ? "OK" : "Error"}
          </div>
          {health() && <div>DB: {health().db_path}</div>}
        </div>
        <button
          onClick={() => setShowDebug(false)}
          class="mt-2 text-xs bg-gray-700 px-2 py-1 rounded hover:bg-gray-600"
        >
          Hide
        </button>
      </div>
    </Show>
  );

  // Show loading state if we haven't attempted session restore yet or if we're trying to restore
  const showLoading = () => {
    // If we have a stored session and haven't restored yet, show loading
    if (
      wsClient.getSessionId() &&
      !currentUser() &&
      !sessionRestoreAttempted()
    ) {
      return true;
    }
    // If we're currently connecting and have a session ID but no user yet, show loading
    if (
      wsClient.getSessionId() &&
      connectionState() === "connecting" &&
      !currentUser()
    ) {
      return true;
    }
    // If we're connected and have a session ID but no user yet (waiting for session_info response)
    if (
      wsClient.getSessionId() &&
      connectionState() === "connected" &&
      !currentUser() &&
      sessionRestoreAttempted()
    ) {
      return true;
    }
    return false;
  };

  const LoadingScreen = () => (
    <div class="min-h-screen bg-gray-900 flex items-center justify-center p-8">
      <div class="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-8 max-w-md w-full text-center">
        <h1 class="text-3xl font-bold text-gray-100 mb-6">ThrowBackChat</h1>
        <div class="space-y-4">
          <div class="flex items-center justify-center">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400"></div>
          </div>
          <div class="text-gray-300">
            {wsClient.getSessionId()
              ? "Restoring your session..."
              : "Loading..."}
          </div>
          <div class="text-sm text-gray-400">
            Connection: {connectionState()}
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <Show when={!showLoading()} fallback={<LoadingScreen />}>
      <Show when={currentUser()} fallback={<Login />}>
        <div class="relative">
          <ChatLayout />
          <DebugOverlay />

          {/* Debug Toggle Button */}
          <button
            onClick={() => setShowDebug(!showDebug())}
            class="fixed bottom-4 right-4 bg-gray-800 text-gray-300 px-3 py-2 rounded-lg text-xs hover:bg-gray-700 transition-colors z-40 border border-gray-600"
          >
            Debug
          </button>
        </div>
      </Show>
    </Show>
  );
}

export default App;
