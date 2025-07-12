import { For, createSignal, Show, createEffect } from "solid-js";
import { currentChannel, getters, chatAPI, appState } from "../store";
import { setCurrentChannel } from "../store";

export function ChannelList() {
  const [isJoining, setIsJoining] = createSignal(false);
  const [joinChannelName, setJoinChannelName] = createSignal("");
  const [showJoinForm, setShowJoinForm] = createSignal(false);

  // Load available channels when user logs in
  createEffect(() => {
    if (getters.isLoggedIn()) {
      chatAPI.listChannels().catch(console.error);
    }
  });

  const handleChannelSelect = async (channelId: string) => {
    // Check if we're actually joined to this channel
    const isJoined = appState.channels[channelId] !== undefined;

    if (!isJoined) {
      // If not joined, this should not happen for joined channels
      // This likely means there's a state issue - the channel shouldn't appear in joined list
      console.error("Channel appears joined but not in state - this is a bug");
      return;
    }

    // At this point we should be joined to the channel
    setCurrentChannel(channelId);

    // Load channel data if not already loaded, and verify we're actually in the channel
    try {
      const channelMessages = appState.messages[channelId] || [];

      // Always try to get channel users to verify we're in the channel
      try {
        await chatAPI.getChannelUsers(channelId);

        // If we successfully got users, load history if needed
        if (channelMessages.length === 0) {
          await chatAPI.getHistory(channelId, undefined, 50);
        }
      } catch (error) {
        // If getting channel users failed, it likely means we're not actually in the channel
        console.log(
          "Failed to get channel users, might not be in channel:",
          error,
        );
        const channel = appState.channels[channelId];
        if (channel) {
          console.log(
            "Attempting to rejoin channel due to state inconsistency:",
            channel.name,
          );
          try {
            // Try to rejoin the channel
            await chatAPI.joinChannel(channel.name);
            console.log("Successfully rejoined channel:", channel.name);

            // Now load the data
            await Promise.all([
              chatAPI.getChannelUsers(channelId),
              chatAPI.getHistory(channelId, undefined, 50),
            ]);
          } catch (rejoinError) {
            console.error("Failed to rejoin channel:", rejoinError);
            // If rejoin fails, remove the stale channel from state
            await chatAPI.leaveChannel(channelId);
          }
        }
      }
    } catch (error) {
      console.error("Failed to load channel data:", error);
    }
  };

  const handleJoinChannel = async (e: Event) => {
    e.preventDefault();
    const channelName = joinChannelName().trim();
    if (!channelName || isJoining()) return;

    setIsJoining(true);
    try {
      await chatAPI.joinChannel(channelName);
      setJoinChannelName("");
      setShowJoinForm(false);
    } catch (error) {
      console.error("Failed to join channel:", error);
    } finally {
      setIsJoining(false);
    }
  };

  const handleLeaveChannel = async (channelId: string, e: Event) => {
    e.stopPropagation();
    try {
      await chatAPI.leaveChannel(channelId);
      // Refresh available channels after leaving
      await chatAPI.listChannels();
    } catch (error) {
      console.error("Failed to leave channel:", error);
    }
  };

  const handleJoinAvailableChannel = async (channelName: string) => {
    try {
      await chatAPI.joinChannel(channelName);
      // Refresh available channels after joining
      await chatAPI.listChannels();
    } catch (error) {
      console.error("Failed to join channel:", error);
    }
  };

  return (
    <div class="h-full flex flex-col">
      {/* Header */}
      <div class="p-4 border-b border-gray-700">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-sm font-semibold text-gray-300 uppercase tracking-wide">
            Channels
          </h2>
          <button
            onClick={() => setShowJoinForm(!showJoinForm())}
            class="text-gray-400 hover:text-gray-200 text-lg leading-none"
            title="Join Channel"
          >
            +
          </button>
        </div>

        {/* Join Channel Form */}
        <Show when={showJoinForm()}>
          <form onSubmit={handleJoinChannel} class="space-y-2">
            <input
              type="text"
              placeholder="Channel name"
              value={joinChannelName()}
              onInput={(e) => setJoinChannelName(e.currentTarget.value)}
              class="w-full px-2 py-1 text-sm bg-gray-700 border border-gray-600 rounded text-gray-100 placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
              disabled={isJoining()}
            />
            <div class="flex space-x-2">
              <button
                type="submit"
                disabled={isJoining() || !joinChannelName().trim()}
                class="flex-1 px-2 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed transition-colors"
              >
                {isJoining() ? "Joining..." : "Join"}
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowJoinForm(false);
                  setJoinChannelName("");
                }}
                class="px-2 py-1 text-xs bg-gray-600 text-gray-300 rounded hover:bg-gray-500 transition-colors"
              >
                Cancel
              </button>
            </div>
          </form>
        </Show>
      </div>

      {/* Channel List */}
      <div class="flex-1 overflow-y-auto">
        {/* Joined Channels Section */}
        <Show
          when={getters.getChannelList().length > 0}
          fallback={
            <div class="p-4 text-sm text-gray-500 text-center">
              No channels joined yet.
              <br />
              Click + to join a channel.
            </div>
          }
        >
          <For each={getters.getChannelList()}>
            {(channel) => (
              <div
                onClick={() => handleChannelSelect(channel.id)}
                class={`
                  group px-4 py-2 cursor-pointer hover:bg-gray-700 transition-colors
                  ${currentChannel() === channel.id ? "bg-blue-600 hover:bg-blue-700" : ""}
                `}
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center space-x-2 min-w-0">
                    <span class="font-medium text-sm truncate">
                      {channel.name}
                    </span>
                  </div>
                  <button
                    onClick={(e) => handleLeaveChannel(channel.id, e)}
                    class="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-red-400 text-xs transition-opacity"
                    title="Leave Channel"
                  >
                    ×
                  </button>
                </div>
                <Show when={channel.topic}>
                  <div class="text-xs text-gray-400 mt-1 truncate">
                    {channel.topic}
                  </div>
                </Show>
              </div>
            )}
          </For>
        </Show>

        {/* Available Channels Section - Always show if there are any */}
        <Show when={getters.getAvailableChannels().length > 0}>
          <div
            class={`border-t border-gray-700 pt-2 ${getters.getChannelList().length > 0 ? "mt-2" : ""}`}
          >
            <div class="px-4 py-2">
              <h3 class="text-xs font-semibold text-gray-500 uppercase tracking-wide">
                Available Channels
              </h3>
            </div>
            <For each={getters.getAvailableChannels()}>
              {(channel) => (
                <div
                  onClick={() => handleJoinAvailableChannel(channel.name)}
                  class="group px-4 py-2 cursor-pointer hover:bg-gray-700 transition-colors"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex items-center space-x-2 min-w-0">
                      <span class="font-medium text-sm truncate text-gray-400">
                        {channel.name}
                      </span>
                      <span class="text-xs text-gray-500">
                        ({channel.user_count})
                      </span>
                    </div>
                    <button
                      class="opacity-0 group-hover:opacity-100 text-gray-500 hover:text-gray-300 text-xs transition-opacity"
                      title="Join Channel"
                    >
                      +
                    </button>
                  </div>
                  <Show when={channel.topic}>
                    <div class="text-xs text-gray-500 mt-1 truncate">
                      {channel.topic}
                    </div>
                  </Show>
                </div>
              )}
            </For>
          </div>
        </Show>
      </div>
    </div>
  );
}
