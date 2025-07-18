import { createStore } from "solid-js/store";
import { createSignal, createEffect, createRoot } from "solid-js";
import { wsClient } from "./websocket";
import type {
  User,
  Channel,
  ChannelInfo,
  Message,
  ChannelUser,
  ConnectionState,
  AppState,
  ChatMessage,
  ChatEvent,
  WebSocketMessage,
  LoginRequest,
  JoinRequest,
  LeaveRequest,
  MessageRequest,
  NickRequest,
  ListChannelsRequest,
  MyChannelsRequest,
  HistoryRequest,
  ChannelUsersRequest,
} from "./types";

// Create the main app store - wrapped in createRoot to prevent disposal warnings
let appState: AppState;
let setAppState: import("solid-js/store").SetStoreFunction<AppState>;

createRoot(() => {
  const result = createStore<AppState>({
    connectionState: "disconnected",
    currentUser: null,
    currentChannel: null,
    channels: {},
    availableChannels: {},
    messages: {},
    channelUsers: {},
    ops: {},
  });
  appState = result[0];
  setAppState = result[1];
});

// Temporary notification system for errors
export interface TempNotification {
  id: string;
  type: "error" | "success" | "info";
  message: string;
  channelId?: string;
  timestamp: number;
}

// Temp notifications store - wrapped in createRoot to prevent disposal warnings
let tempNotifications: TempNotification[];
let setTempNotifications: import("solid-js/store").SetStoreFunction<
  TempNotification[]
>;

createRoot(() => {
  const result = createStore<TempNotification[]>([]);
  tempNotifications = result[0];
  setTempNotifications = result[1];
});

let notificationTimeouts = new Map<string, number>();

export const showTempNotification = (
  type: "error" | "success" | "info",
  message: string,
  channelId?: string,
  duration = 5000,
) => {
  const id = `notif_${Date.now()}_${Math.random()}`;
  const notification: TempNotification = {
    id,
    type,
    message,
    channelId,
    timestamp: Date.now(),
  };

  setTempNotifications((prev) => [...prev, notification]);

  // Auto-remove after duration
  const timeoutId = setTimeout(() => {
    setTempNotifications((prev) => prev.filter((n) => n.id !== id));
    notificationTimeouts.delete(id);
  }, duration);

  notificationTimeouts.set(id, timeoutId);
};

export const removeTempNotification = (id: string) => {
  const timeoutId = notificationTimeouts.get(id);
  if (timeoutId) {
    clearTimeout(timeoutId);
    notificationTimeouts.delete(id);
  }
  setTempNotifications((prev) => prev.filter((n) => n.id !== id));
};

export const getTempNotifications = () => tempNotifications;

// Create reactive signals for commonly accessed state - wrapped in createRoot to prevent disposal warnings
let connectionState: () => ConnectionState;
let setConnectionState: (value: ConnectionState) => void;
let currentUser: () => User | null;
let setCurrentUser: (
  value: User | null | ((prev: User | null) => User | null),
) => void;
let currentChannel: () => string | null;
let setCurrentChannel: (value: string | null) => void;

createRoot(() => {
  const connectionStateResult = createSignal<ConnectionState>("disconnected");
  connectionState = connectionStateResult[0];
  setConnectionState = connectionStateResult[1];

  const currentUserResult = createSignal<User | null>(null);
  currentUser = currentUserResult[0];
  setCurrentUser = currentUserResult[1];

  const currentChannelResult = createSignal<string | null>(null);
  currentChannel = currentChannelResult[0];
  setCurrentChannel = currentChannelResult[1];
});

// Sync signals with store - wrapped in createRoot to prevent disposal warnings
createRoot(() => {
  createEffect(() => {
    setAppState("connectionState", connectionState());
  });

  createEffect(() => {
    setAppState("currentUser", currentUser());
  });

  createEffect(() => {
    setAppState("currentChannel", currentChannel());
  });
});

// WebSocket message handler
function handleWebSocketMessage(message: WebSocketMessage) {
  console.log("Received WebSocket message:", message);

  switch (message.type) {
    case "message":
      handleChatMessage(message as ChatMessage);
      break;
    case "event":
      handleChatEvent(message as ChatEvent);
      break;
    case "response":
      // Response messages are handled by the WebSocket client
      break;
    default:
      console.warn("Unknown message type:", message);
  }
}

function handleChatMessage(message: ChatMessage) {
  const channelId = message.channel_id;

  // Add message to channel's message list
  setAppState("messages", channelId, (messages = []) => [
    ...messages,
    {
      id: `msg_${Date.now()}_${Math.random()}`,
      channel_id: channelId,
      user_id: message.user_id,
      nickname: message.nickname,
      message: message.message,
      is_passive: message.is_passive,
      event: "message",
      sent_at: message.sent_at,
    },
  ]);

  // Limit message history to 1000 messages per channel
  setAppState("messages", channelId, (messages = []) =>
    messages.length > 1000 ? messages.slice(-1000) : messages,
  );
}

function handleChatEvent(event: ChatEvent) {
  const channelId = event.channel_id;

  // Create a message representation of the event
  const eventMessage: Message = {
    id: `event_${Date.now()}_${Math.random()}`,
    channel_id: channelId || "server",
    user_id: event.user_id || "system",
    nickname: event.nickname || "System",
    message: formatEventMessage(event),
    is_passive: true,
    event: event.event,
    sent_at: event.sent_at,
  };

  // Add event as message
  if (channelId) {
    setAppState("messages", channelId, (messages = []) => [
      ...messages,
      eventMessage,
    ]);
  }

  // Handle specific event types - for real-time user updates, refresh the channel users list
  switch (event.event) {
    case "joined":
      if (channelId) {
        // Refresh channel users when someone joins
        refreshChannelUsers(channelId);
        // Also refresh the available channels list to update user counts
        refreshChannelList();
      }
      break;

    case "left":
      if (channelId) {
        // Check if the person who left was the current user
        if (event.user_id === currentUser()?.id) {
          // If I left the channel, remove it from my state completely
          setAppState("channels", (channels) => {
            const { [channelId]: removed, ...rest } = channels;
            return rest;
          });

          setAppState("messages", (messages) => {
            const { [channelId]: removed, ...rest } = messages;
            return rest;
          });

          setAppState("channelUsers", (channelUsers) => {
            const { [channelId]: removed, ...rest } = channelUsers;
            return rest;
          });

          // If this was the current channel, clear it
          if (currentChannel() === channelId) {
            setCurrentChannel(null);
          }
        } else {
          // Someone else left, just refresh channel users
          refreshChannelUsers(channelId);
        }
        // Always refresh the available channels list to update user counts
        refreshChannelList();
      }
      break;

    case "nick_change":
      if (event.user_id && event.new_nickname) {
        // Update nickname in current user if it's us
        if (currentUser()?.id === event.user_id) {
          setCurrentUser((prev) =>
            prev ? { ...prev, nickname: event.new_nickname! } : null,
          );
        }

        // Refresh channel users to get updated nicknames
        if (channelId) {
          refreshChannelUsers(channelId);
        }
      }
      break;

    case "topic_change":
      if (channelId && event.topic !== undefined) {
        // Ensure channel exists in store before setting topic
        setAppState("channels", channelId, (channel) => ({
          id: channelId.toString(),
          name: channel?.name || `#${channelId}`,
          topic: event.topic,
          ...channel,
        }));
      }
      break;
  }
}

function formatEventMessage(event: ChatEvent): string {
  switch (event.event) {
    case "joined":
      return `${event.nickname} joined the channel`;
    case "left":
      return `${event.nickname} left the channel`;
    case "nick_change":
      return `${event.old_nickname} is now known as ${event.new_nickname}`;
    case "kicked":
      return `${event.nickname} was kicked${event.kicked_by ? ` by ${event.kicked_by}` : ""}${event.reason ? `: ${event.reason}` : ""}`;
    case "topic_change":
      return `Topic changed to: ${event.topic}`;
    case "announcement":
      return event.message || "Server announcement";
    default:
      return `Unknown event: ${event.event}`;
  }
}

// Setup WebSocket connection and message handling
wsClient.onMessage = handleWebSocketMessage;
wsClient.onConnectionChange = (state: ConnectionState) => {
  setConnectionState(state);

  // Clear state on disconnect only if not attempting reconnection
  if (state === "disconnected" || state === "error") {
    // Don't clear user state if we're just reconnecting
    if (state === "error") {
      // Keep session data for reconnection
      return;
    }

    setCurrentUser(null);
    setCurrentChannel(null);
    setAppState({
      connectionState: state,
      currentUser: null,
      currentChannel: null,
      channels: {},
      availableChannels: {},
      messages: {},
      channelUsers: {},
      ops: {},
    });
  }

  // Attempt session restoration on successful connection
  if (state === "connected") {
    attemptSessionRestoration();
  }
};

// Attempt to restore session after connection
async function attemptSessionRestoration() {
  const sessionId = wsClient.getSessionId();
  if (!sessionId) {
    console.log("No session ID to restore");
    return;
  }

  try {
    console.log("Attempting to restore session:", sessionId);
    // Try to get session info from server
    const response = await wsClient.send<any>({
      cmd: "session_info",
      req_id: "",
    });

    if (response.okay && response.data) {
      // Restore user state
      if (response.data.user_id && response.data.nickname) {
        console.log(
          "Session restored successfully for user:",
          response.data.nickname,
        );
        setCurrentUser({
          id: response.data.user_id,
          nickname: response.data.nickname,
          is_serv: false,
        });

        // Restore channels and load their full state
        if (response.data.channels && response.data.channels.length > 0) {
          const restoredChannels = response.data.channels;

          // First, add channels to state
          restoredChannels.forEach((channel: any) => {
            setAppState("channels", channel.id.toString(), {
              id: channel.id.toString(),
              name: channel.name,
              topic: channel.topic || "",
            });
          });

          // Load full state for each channel (users and messages)
          for (const channel of restoredChannels) {
            try {
              // Load channel users and message history in parallel
              const [usersResponse, historyResponse] = await Promise.all([
                wsClient.send<any>({
                  cmd: "channel_users",
                  channel_id: channel.id,
                  req_id: "",
                }),
                wsClient.send<any>({
                  cmd: "get_history",
                  channel_id: channel.id,
                  limit: 50,
                  req_id: "",
                }),
              ]);

              // Set channel users
              if (usersResponse.okay) {
                setAppState(
                  "channelUsers",
                  channel.id.toString(),
                  usersResponse.data.users || [],
                );
              }

              // Set channel messages
              if (historyResponse.okay) {
                const messages = (historyResponse.data.messages || []).map(
                  (msg: any) => {
                    if (msg.type === "event") {
                      return {
                        id: `event_${Date.now()}_${Math.random()}`,
                        channel_id: channel.id.toString(),
                        user_id: msg.user_id || "system",
                        nickname: msg.nickname || "System",
                        message: formatEventMessage(msg),
                        is_passive: true,
                        event: msg.event,
                        sent_at: msg.sent_at,
                      };
                    } else {
                      return {
                        id: `msg_${Date.now()}_${Math.random()}`,
                        channel_id: channel.id.toString(),
                        user_id: msg.user_id,
                        nickname: msg.nickname,
                        message: msg.message,
                        is_passive: msg.is_passive,
                        event: "message",
                        sent_at: msg.sent_at,
                      };
                    }
                  },
                );
                setAppState("messages", channel.id.toString(), messages);
              }
            } catch (error) {
              console.error(
                `Failed to load state for channel ${channel.name}:`,
                error,
              );
            }
          }

          // Set the first channel as current channel if none is set
          if (!currentChannel() && restoredChannels.length > 0) {
            setCurrentChannel(restoredChannels[0].id.toString());
            console.log("Set current channel to:", restoredChannels[0].name);
          }
        }
      } else {
        console.log("Session exists but user not logged in");
        wsClient.clearSession();
      }
    } else {
      console.log(
        "Session restoration failed:",
        response.error || "Unknown error",
      );
      wsClient.clearSession();
    }
  } catch (error) {
    console.log("Session restoration failed:", error);
    // Clear stored session if restoration fails
    wsClient.clearSession();
  }
}

// Helper function to refresh channel users
async function refreshChannelUsers(channelId: string) {
  try {
    const users = await chatAPI.getChannelUsers(channelId);
    setAppState("channelUsers", channelId, users);
  } catch (error) {
    console.error("Failed to refresh channel users:", error);
  }
}

// Helper function to refresh available channels list
async function refreshChannelList() {
  try {
    await chatAPI.listChannels();
  } catch (error) {
    console.error("Failed to refresh channel list:", error);
  }
}

// Chat API functions
export const chatAPI = {
  async login(nickname: string): Promise<void> {
    const response = await wsClient.send<any>({
      cmd: "login",
      nickname,
      req_id: "",
    } as LoginRequest);

    if (response.okay) {
      setCurrentUser({
        id: response.data.user_id,
        nickname: response.data.nickname || nickname,
        is_serv: false,
      });

      // Store session ID after successful login
      if (response.data.session_id) {
        wsClient.setSessionId(response.data.session_id);
      }

      // Automatically fetch user's channels after login
      try {
        const myChannelsResponse = await wsClient.send<any>({
          cmd: "my_channels",
          req_id: "",
        });

        if (myChannelsResponse.okay) {
          const channels = myChannelsResponse.data.channels || [];
          channels.forEach((channel: any) => {
            setAppState("channels", channel.id, channel);
          });
        }
      } catch (error) {
        console.error("Failed to fetch channels after login:", error);
      }
    } else {
      throw new Error(response.error || "Login failed");
    }
  },

  async logout(dyingMessage?: string): Promise<void> {
    await wsClient.send({
      cmd: "logout",
      dying_message: dyingMessage,
      req_id: "",
    });

    wsClient.disconnect();
  },

  async quit(dyingMessage?: string): Promise<void> {
    try {
      await wsClient.send({
        cmd: "quit",
        dying_message: dyingMessage,
        req_id: "",
      });
    } catch (error) {
      console.log("Quit command failed, forcing disconnect:", error);
    }

    // Clear session storage and disconnect
    wsClient.disconnect(true); // true = clear session
  },

  async joinChannel(channelName: string): Promise<void> {
    const response = await wsClient.send<any>({
      cmd: "join",
      channel_name: channelName,
      req_id: "",
    } as JoinRequest);

    if (response.okay) {
      const channel: Channel = {
        id: response.data.channel_id.toString(),
        name: response.data.channel_name,
        topic: "",
      };

      setAppState("channels", response.data.channel_id.toString(), channel);
      setCurrentChannel(response.data.channel_id.toString());

      // Load message history and channel users for the channel
      try {
        const [{ messages }, users] = await Promise.all([
          this.getHistory(response.data.channel_id.toString(), undefined, 50),
          this.getChannelUsers(response.data.channel_id.toString()),
        ]);
        setAppState("messages", response.data.channel_id.toString(), messages);
        setAppState("channelUsers", response.data.channel_id.toString(), users);
      } catch (error) {
        console.error("Failed to load channel data:", error);
        // Don't fail the join if loading fails
      }
    } else {
      throw new Error(response.error || "Failed to join channel");
    }
  },

  async leaveChannel(channelId: string): Promise<void> {
    const response = await wsClient.send<any>({
      cmd: "leave",
      channel_id: parseInt(channelId),
      req_id: "",
    } as LeaveRequest);

    if (response.okay) {
      setAppState("channels", (channels) => {
        const { [channelId]: removed, ...rest } = channels;
        return rest;
      });

      setAppState("messages", (messages) => {
        const { [channelId]: removed, ...rest } = messages;
        return rest;
      });

      setAppState("channelUsers", (channelUsers) => {
        const { [channelId]: removed, ...rest } = channelUsers;
        return rest;
      });

      if (currentChannel() === channelId) {
        setCurrentChannel(null);
      }
    } else {
      throw new Error(response.error || "Failed to leave channel");
    }
  },

  async sendMessage(
    channelId: string,
    message: string,
    isPassive = false,
  ): Promise<void> {
    const response = await wsClient.send({
      cmd: "message",
      channel_id: parseInt(channelId, 10),
      message,
      is_passive: isPassive,
      req_id: "",
    } as MessageRequest);

    if (!response.okay) {
      throw new Error(response.error || "Failed to send message");
    }
  },

  async changeNickname(newNickname: string): Promise<void> {
    const response = await wsClient.send({
      cmd: "nick",
      new_nickname: newNickname,
      req_id: "",
    } as NickRequest);

    if (!response.okay) {
      throw new Error(response.error || "Failed to change nickname");
    }
  },

  async kickUser(
    channelId: string,
    userId: string,
    reason?: string,
  ): Promise<void> {
    const response = await wsClient.send({
      cmd: "kick",
      channel_id: parseInt(channelId, 10),
      user_id: parseInt(userId, 10),
      reason,
      req_id: "",
    });

    if (!response.okay) {
      throw new Error(response.error || "Failed to kick user");
    }
  },

  async changeTopic(channelId: string, topic: string): Promise<void> {
    const response = await wsClient.send({
      cmd: "topic",
      channel_id: parseInt(channelId, 10),
      topic,
      req_id: "",
    });

    if (!response.okay) {
      throw new Error(response.error || "Failed to change topic");
    }
  },

  async sendMeAction(channelId: string, message: string): Promise<void> {
    const response = await wsClient.send({
      cmd: "me",
      channel_id: parseInt(channelId, 10),
      message,
      req_id: "",
    });

    if (!response.okay) {
      throw new Error(response.error || "Failed to send action");
    }
  },

  async announce(message: string, channelId?: string): Promise<void> {
    const response = await wsClient.send({
      cmd: "announce",
      channel_id: channelId ? parseInt(channelId) : undefined,
      message,
      req_id: "",
    });

    if (!response.okay) {
      throw new Error(response.error || "Failed to send announcement");
    }
  },

  async listChannels(): Promise<ChannelInfo[]> {
    const response = await wsClient.send<any>({
      cmd: "list_channels",
      req_id: "",
    } as ListChannelsRequest);

    if (response.okay) {
      const channels = response.data?.channels || [];

      // Update available channels in store (only those with users)
      const channelsWithUsers = channels.filter(
        (ch: ChannelInfo) => ch.user_count > 0,
      );
      const availableChannelsObj = channelsWithUsers.reduce(
        (acc: Record<string, ChannelInfo>, ch: ChannelInfo) => {
          acc[ch.id.toString()] = ch;
          return acc;
        },
        {},
      );

      setAppState("availableChannels", availableChannelsObj);

      return channels;
    } else {
      throw new Error(response.error || "Failed to list channels");
    }
  },

  async getMyChannels(): Promise<Channel[]> {
    const response = await wsClient.send<any>({
      cmd: "my_channels",
      req_id: "",
    } as MyChannelsRequest);

    if (response.okay) {
      const channels = response.data?.channels || [];

      // Update store with channels
      channels.forEach((channel: Channel) => {
        setAppState("channels", channel.id, channel);
      });

      return channels;
    } else {
      throw new Error(response.error || "Failed to get my channels");
    }
  },

  async getChannelUsers(channelId: string): Promise<ChannelUser[]> {
    const response = await wsClient.send<any>({
      cmd: "channel_users",
      channel_id: parseInt(channelId),
      req_id: "",
    } as ChannelUsersRequest);

    if (response.okay) {
      return response.data?.users || [];
    } else {
      throw new Error(response.error || "Failed to get channel users");
    }
  },

  async getHistory(
    channelId: string,
    before?: number,
    limit = 50,
  ): Promise<{ messages: Message[]; hasMore: boolean }> {
    const response = await wsClient.send<any>({
      cmd: "get_history",
      channel_id: parseInt(channelId),
      before,
      limit,
      req_id: "",
    } as HistoryRequest);

    if (response.okay) {
      const messages = (response.data?.messages || []).map((msg: any) => {
        if (msg.type === "event") {
          return {
            id: `event_${Date.now()}_${Math.random()}`,
            channel_id: channelId,
            user_id: msg.user_id || "system",
            nickname: msg.nickname || "System",
            message: formatEventMessage(msg as ChatEvent),
            is_passive: true,
            event: msg.event,
            sent_at: msg.sent_at,
          };
        } else {
          return {
            id: `msg_${Date.now()}_${Math.random()}`,
            channel_id: channelId,
            user_id: msg.user_id,
            nickname: msg.nickname,
            message: msg.message,
            is_passive: msg.is_passive,
            event: "message",
            sent_at: msg.sent_at,
          };
        }
      });

      return {
        messages,
        hasMore: response.data?.has_more || false,
      };
    } else {
      throw new Error(response.error || "Failed to get message history");
    }
  },

  connect(): void {
    wsClient.connect();
  },

  disconnect(): void {
    wsClient.disconnect();
  },
};

// Export the store and reactive accessors
export {
  appState,
  connectionState,
  currentUser,
  currentChannel,
  setCurrentChannel,
};

// Computed getters for convenience
export const getters = {
  isConnected: () => connectionState() === "connected",
  isLoggedIn: () => currentUser() !== null,
  getCurrentChannelData: () => {
    const channelId = currentChannel();
    return channelId ? appState.channels[channelId] : null;
  },
  getCurrentChannelMessages: () => {
    const channelId = currentChannel();
    return channelId ? appState.messages[channelId] || [] : [];
  },
  getCurrentChannelUsers: () => {
    const channelId = currentChannel();
    return channelId ? appState.channelUsers[channelId] || [] : [];
  },
  getChannelList: () =>
    Object.values(appState.channels).sort((a, b) =>
      a.name.localeCompare(b.name),
    ),
  getAvailableChannels: () => {
    const joinedChannelIds = new Set(Object.keys(appState.channels));
    return Object.values(appState.availableChannels)
      .filter((ch) => !joinedChannelIds.has(ch.id.toString()))
      .sort((a, b) => a.name.localeCompare(b.name));
  },
};
