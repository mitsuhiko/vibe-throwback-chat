import { createSignal, onMount } from "solid-js";

function App() {
  const [health, setHealth] = createSignal<any>(null);
  const [loading, setLoading] = createSignal(true);

  onMount(async () => {
    try {
      const response = await fetch("/api/health");
      const data = await response.json();
      setHealth(data);
    } catch (error) {
      console.error("Failed to fetch health:", error);
    } finally {
      setLoading(false);
    }
  });

  return (
    <div class="min-h-screen bg-gray-100 flex items-center justify-center p-8">
      <div class="bg-white rounded-lg shadow-lg p-8 max-w-md w-full">
        <h1 class="text-3xl font-bold text-gray-900 mb-6 text-center">
          ThrowBackChat
        </h1>

        <div class="space-y-4">
          <div class="text-center">
            <h2 class="text-xl font-semibold text-gray-700 mb-2">
              Server Status
            </h2>

            {loading() ? (
              <div class="text-gray-500">Loading...</div>
            ) : health() ? (
              <div class="space-y-2">
                <div class="text-green-600 font-semibold">
                  ✓ Server is running
                </div>
                <div class="text-sm text-gray-600">
                  Database: {health().db_path}
                </div>
              </div>
            ) : (
              <div class="text-red-600 font-semibold">
                ✗ Server is not responding
              </div>
            )}
          </div>

          <div class="text-center text-sm text-gray-500">
            Initial setup complete. Chat interface coming soon!
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
