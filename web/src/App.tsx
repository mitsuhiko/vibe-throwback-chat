import { createSignal, createEffect, Show } from "solid-js";
import { currentUser, connectionState } from "./store";
import { Login } from "./components/Login";
import { ChatLayout } from "./components/ChatLayout";

function App() {
  const [health, setHealth] = createSignal<any>(null);
  const [healthLoading, setHealthLoading] = createSignal(true);
  const [showDebug, setShowDebug] = createSignal(false);

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


  const DebugOverlay = () => (
    <Show when={showDebug()}>
      <div class="absolute top-4 right-4 bg-black bg-opacity-75 text-white p-4 rounded-lg text-xs z-50 max-w-xs">
        <h3 class="font-semibold mb-2">Debug Information</h3>
        <div class="space-y-1">
          <div>User ID: {currentUser()?.id}</div>
          <div>Nickname: {currentUser()?.nickname}</div>
          <div>Connection: {connectionState()}</div>
          <div>Health: {healthLoading() ? "Loading..." : health() ? "OK" : "Error"}</div>
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

  return (
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
  );
}

export default App;
