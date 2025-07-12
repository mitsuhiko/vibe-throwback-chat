import { Show } from "solid-js";
import { currentUser, currentChannel, getters, chatAPI } from "../store";
import { ChannelList } from "./ChannelList";
import { ChatArea } from "./ChatArea";
import { UserList } from "./UserList";
import { MessageInput } from "./MessageInput";

export function ChatLayout() {
  const handleLogout = async () => {
    try {
      await chatAPI.logout();
    } catch (error) {
      console.error("Logout failed:", error);
    }
  };

  return (
    <div class="h-screen bg-gray-900 text-gray-100 flex flex-col">
      {/* Header */}
      <header class="bg-gray-800 border-b border-gray-700 px-4 py-3 flex items-center justify-between">
        <div class="flex items-center space-x-4">
          <h1 class="text-xl font-bold text-white">ThrowBackChat</h1>
          <Show when={getters.getCurrentChannelData()}>
            <div class="flex items-center space-x-2">
              <span class="text-gray-400">#</span>
              <span class="font-medium">
                {getters.getCurrentChannelData()?.name}
              </span>
              <Show when={getters.getCurrentChannelData()?.topic}>
                <span class="text-gray-400">-</span>
                <span class="text-sm text-gray-400">
                  {getters.getCurrentChannelData()?.topic}
                </span>
              </Show>
            </div>
          </Show>
        </div>
        <div class="flex items-center space-x-4">
          <span class="text-sm text-gray-400">{currentUser()?.nickname}</span>
          <button
            onClick={handleLogout}
            class="px-3 py-1 text-sm bg-gray-700 text-gray-300 rounded hover:bg-gray-600 transition-colors"
          >
            Logout
          </button>
        </div>
      </header>

      {/* Main Content */}
      <div class="flex-1 flex overflow-hidden">
        {/* Left Sidebar - Channel List */}
        <div class="w-60 bg-gray-800 border-r border-gray-700 flex-shrink-0">
          <ChannelList />
        </div>

        {/* Center - Chat Area */}
        <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
          <Show
            when={currentChannel()}
            fallback={
              <div class="flex-1 flex items-center justify-center bg-gray-900">
                <div class="text-center">
                  <h2 class="text-xl font-semibold text-gray-400 mb-2">
                    Welcome to ThrowBackChat
                  </h2>
                  <p class="text-gray-500">
                    Select a channel from the sidebar or join a new one to start chatting.
                    <br />
                    You can also use commands like <span class="font-mono text-gray-300">/join #channelname</span> below.
                  </p>
                </div>
              </div>
            }
          >
            <div class="flex-1 overflow-hidden">
              <ChatArea />
            </div>
          </Show>
          
          {/* Message Input - Always visible */}
          <div class="flex-shrink-0">
            <MessageInput />
          </div>
        </div>

        {/* Right Sidebar - User List */}
        <Show when={currentChannel()}>
          <div class="w-48 bg-gray-800 border-l border-gray-700 flex-shrink-0">
            <UserList />
          </div>
        </Show>
      </div>
    </div>
  );
}
