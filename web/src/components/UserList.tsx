import { For, Show, createSignal, onMount, onCleanup, createEffect } from "solid-js";
import { getters, currentUser, chatAPI, connectionState } from "../store";
import type { ChannelUser } from "../types";

export function UserList() {
  const users = () => getters.getCurrentChannelUsers();
  const userCount = () => users().length;
  const [contextMenu, setContextMenu] = createSignal<{ x: number, y: number, user: ChannelUser } | null>(null);
  const [selectedUser, setSelectedUser] = createSignal<ChannelUser | null>(null);
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  
  const isConnected = () => connectionState() === "connected";
  const currentChannelData = () => getters.getCurrentChannelData();

  // Close context menu on outside click
  onMount(() => {
    const handleClick = () => setContextMenu(null);
    document.addEventListener('click', handleClick);
    onCleanup(() => document.removeEventListener('click', handleClick));
  });

  const handleUserClick = (user: ChannelUser) => {
    // Insert mention in chat input
    const chatInput = document.querySelector('input[placeholder="Type a message..."]') as HTMLInputElement;
    if (chatInput) {
      const mention = `@${user.nickname} `;
      const currentValue = chatInput.value;
      const cursorPos = chatInput.selectionStart || 0;
      const newValue = currentValue.slice(0, cursorPos) + mention + currentValue.slice(cursorPos);
      chatInput.value = newValue;
      chatInput.focus();
      chatInput.setSelectionRange(cursorPos + mention.length, cursorPos + mention.length);
    }
  };

  const handleUserRightClick = (e: MouseEvent, user: ChannelUser) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, user });
  };

  const handleKickUser = async (user: ChannelUser) => {
    const channelId = currentChannelData()?.id;
    if (channelId && isConnected()) {
      try {
        setIsLoading(true);
        setError(null);
        await chatAPI.kickUser(channelId, user.id.toString());
        setContextMenu(null);
      } catch (error) {
        console.error('Failed to kick user:', error);
        setError(`Failed to kick ${user.nickname}: ${error instanceof Error ? error.message : 'Unknown error'}`);
      } finally {
        setIsLoading(false);
      }
    } else if (!isConnected()) {
      setError('Cannot kick user: Not connected to server');
    }
  };

  const getUserStatusColor = (user: ChannelUser) => {
    if (user.is_serv) return 'bg-yellow-600';
    if (user.is_op) return 'bg-blue-600';
    return 'bg-green-600';
  };

  const getUserStatusIcon = (user: ChannelUser) => {
    if (user.is_serv) return '⚙';
    if (user.is_op) return '@';
    return user.nickname.charAt(0).toUpperCase();
  };

  const canKickUser = (user: ChannelUser) => {
    const current = currentUser();
    if (!current) return false;
    
    // Find current user in channel users to check if they're an op
    const currentChannelUser = users().find(u => u.id.toString() === current.id);
    if (!currentChannelUser?.is_op && !current.is_serv) return false;
    
    // Can't kick yourself, service users, or other ops (unless you're a service user)
    if (user.id.toString() === current.id) return false;
    if (user.is_serv && !current.is_serv) return false;
    if (user.is_op && !current.is_serv) return false;
    
    return true;
  };

  return (
    <div class="h-full flex flex-col">
      {/* Header */}
      <div class="p-4 border-b border-gray-700">
        <h2 class="text-sm font-semibold text-gray-300 uppercase tracking-wide">
          Users ({userCount()})
        </h2>
        <Show when={!isConnected()}>
          <div class="text-xs text-red-400 mt-1">Disconnected</div>
        </Show>
      </div>

      {/* Error Display */}
      <Show when={error()}>
        <div class="p-3 bg-red-900 border-b border-red-700">
          <div class="text-sm text-red-200">{error()}</div>
          <button 
            class="text-xs text-red-400 hover:text-red-300 mt-1"
            onClick={() => setError(null)}
          >
            Dismiss
          </button>
        </div>
      </Show>

      {/* User List */}
      <div class="flex-1 overflow-y-auto">
        <Show
          when={users().length > 0}
          fallback={
            <div class="p-4 text-sm text-gray-500 text-center">
              <Show when={!isConnected()} fallback="No users in this channel">
                Connect to see channel users
              </Show>
            </div>
          }
        >
          <div class="py-2">
            <For each={users()}>
              {(user) => (
                <div 
                  class={`
                    group px-4 py-2 flex items-center space-x-3 hover:bg-gray-700 transition-colors cursor-pointer relative
                    ${user.id.toString() === currentUser()?.id ? "bg-gray-700 bg-opacity-50" : ""}
                  `}
                  onClick={() => handleUserClick(user)}
                  onContextMenu={(e) => handleUserRightClick(e, user)}
                  title={`Click to mention ${user.nickname}${user.is_op ? ' (Operator)' : ''}${user.is_serv ? ' (Service)' : ''}`}
                >
                  {/* User Status Indicator */}
                  <div class="flex-shrink-0">
                    <div class="relative">
                      <div class={`w-8 h-8 ${getUserStatusColor(user)} rounded-full flex items-center justify-center`}>
                        <span class="text-sm font-medium text-white">
                          {getUserStatusIcon(user)}
                        </span>
                      </div>
                      {/* Online status dot */}
                      <div class="absolute -bottom-1 -right-1 w-3 h-3 bg-green-400 border-2 border-gray-800 rounded-full"></div>
                      {/* Operator badge */}
                      <Show when={user.is_op && !user.is_serv}>
                        <div class="absolute -top-1 -right-1 w-4 h-4 bg-blue-500 border-2 border-gray-800 rounded-full flex items-center justify-center">
                          <span class="text-xs text-white font-bold">@</span>
                        </div>
                      </Show>
                    </div>
                  </div>

                  {/* User Info */}
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center space-x-2">
                      <span class={`font-medium text-sm truncate ${
                        user.is_serv ? 'text-yellow-300' : 
                        user.is_op ? 'text-blue-300' : 
                        'text-gray-200'
                      }`}>
                        {user.nickname}
                      </span>
                      <Show when={user.id.toString() === currentUser()?.id}>
                        <span class="text-xs text-green-400">(you)</span>
                      </Show>
                      <Show when={user.is_serv}>
                        <span class="text-xs text-yellow-400 font-bold" title="Service User">★</span>
                      </Show>
                      <Show when={user.is_op && !user.is_serv}>
                        <span class="text-xs text-blue-400 font-bold" title="Channel Operator">@</span>
                      </Show>
                    </div>
                    <div class="text-xs text-gray-400">
                      <Show when={user.is_serv} fallback="Online">
                        Service
                      </Show>
                      <Show when={user.is_op && !user.is_serv}>
                        {" • Operator"}
                      </Show>
                    </div>
                  </div>

                  {/* User Actions Menu */}
                  <div class="flex-shrink-0">
                    <Show when={user.id.toString() !== currentUser()?.id && canKickUser(user)}>
                      <button 
                        class="opacity-0 group-hover:opacity-100 text-gray-500 hover:text-gray-300 text-xs p-1 transition-opacity"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleUserRightClick(e, user);
                        }}
                        title="User actions"
                      >
                        ⋯
                      </button>
                    </Show>
                  </div>
                </div>
              )}
            </For>
          </div>
        </Show>
      </div>

      {/* Context Menu */}
      <Show when={contextMenu()}>
        <div 
          class="fixed bg-gray-800 border border-gray-600 rounded-lg shadow-lg py-2 z-50"
          style={{ 
            left: `${contextMenu()!.x}px`, 
            top: `${contextMenu()!.y}px` 
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <button
            class="w-full px-4 py-2 text-left text-sm text-gray-200 hover:bg-gray-700 transition-colors"
            onClick={() => {
              const user = contextMenu()!.user;
              handleUserClick(user);
              setContextMenu(null);
            }}
          >
            Mention {contextMenu()!.user.nickname}
          </button>
          <Show when={canKickUser(contextMenu()!.user)}>
            <button
              class="w-full px-4 py-2 text-left text-sm text-red-400 hover:bg-gray-700 transition-colors"
              onClick={() => handleKickUser(contextMenu()!.user)}
            >
              Kick {contextMenu()!.user.nickname}
            </button>
          </Show>
          <Show when={contextMenu()!.user.is_op}>
            <div class="px-4 py-1 text-xs text-gray-500 border-t border-gray-600 mt-1">
              Channel Operator
            </div>
          </Show>
          <Show when={contextMenu()!.user.is_serv}>
            <div class="px-4 py-1 text-xs text-gray-500 border-t border-gray-600 mt-1">
              Service User
            </div>
          </Show>
        </div>
      </Show>

      {/* Footer with channel info */}
      <Show when={getters.getCurrentChannelData()}>
        <div class="p-3 border-t border-gray-700 bg-gray-800">
          <div class="text-xs text-gray-400">
            <div class="font-medium">#{getters.getCurrentChannelData()?.name}</div>
            <Show when={getters.getCurrentChannelData()?.topic}>
              <div class="mt-1 truncate" title={getters.getCurrentChannelData()?.topic}>
                {getters.getCurrentChannelData()?.topic}
              </div>
            </Show>
          </div>
        </div>
      </Show>
    </div>
  );
}