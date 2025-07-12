import { For, createSignal, Show } from "solid-js";
import { currentChannel, getters, chatAPI } from "../store";
import { setCurrentChannel } from "../store";

export function ChannelList() {
  const [isJoining, setIsJoining] = createSignal(false);
  const [joinChannelName, setJoinChannelName] = createSignal("");
  const [showJoinForm, setShowJoinForm] = createSignal(false);

  const handleChannelSelect = (channelId: string) => {
    setCurrentChannel(channelId);
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
    } catch (error) {
      console.error("Failed to leave channel:", error);
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
                    <span class="text-gray-400">#</span>
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
      </div>
    </div>
  );
}