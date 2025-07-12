// Base message interfaces
export interface BaseRequest {
  cmd: string;
  req_id: string;
}

export interface BaseResponse {
  type: "response";
  req_id: string;
  okay: boolean;
  error?: string;
}

// Request types
export interface LoginRequest extends BaseRequest {
  cmd: "login";
  nickname: string;
}

export interface LogoutRequest extends BaseRequest {
  cmd: "logout";
  dying_message?: string;
}

export interface JoinRequest extends BaseRequest {
  cmd: "join";
  channel_name?: string;
  channel_id?: string;
}

export interface LeaveRequest extends BaseRequest {
  cmd: "leave";
  channel_name?: string;
  channel_id?: number;
}

export interface MessageRequest extends BaseRequest {
  cmd: "message";
  channel_id: number;
  message: string;
  is_passive?: boolean;
}

export interface NickRequest extends BaseRequest {
  cmd: "nick";
  new_nickname: string;
}

export interface KickRequest extends BaseRequest {
  cmd: "kick";
  user_id: string;
  channel_id: string;
  reason?: string;
}

export interface TopicRequest extends BaseRequest {
  cmd: "topic";
  channel_id: string;
  topic: string;
}

export interface MeRequest extends BaseRequest {
  cmd: "me";
  channel_id: string;
  message: string;
}

export interface AnnounceRequest extends BaseRequest {
  cmd: "announce";
  channel_id?: number;
  message: string;
}

export interface HeartbeatRequest extends BaseRequest {
  cmd: "heartbeat";
}

export interface ListChannelsRequest extends BaseRequest {
  cmd: "list_channels";
}

export interface MyChannelsRequest extends BaseRequest {
  cmd: "my_channels";
}

export interface HistoryRequest extends BaseRequest {
  cmd: "get_history";
  channel_id: number;
  before?: number;
  after?: number;
  limit?: number;
}

export interface ChannelUsersRequest extends BaseRequest {
  cmd: "channel_users";
  channel_id: number;
}

export type WebSocketRequest =
  | LoginRequest
  | LogoutRequest
  | JoinRequest
  | LeaveRequest
  | MessageRequest
  | NickRequest
  | KickRequest
  | TopicRequest
  | MeRequest
  | AnnounceRequest
  | HeartbeatRequest
  | ListChannelsRequest
  | MyChannelsRequest
  | HistoryRequest
  | ChannelUsersRequest;

// Response types
export interface LoginResponse extends BaseResponse {
  user_id?: string;
  nickname?: string;
}

export interface JoinResponse extends BaseResponse {
  channel_id?: string;
  channel_name?: string;
}

export interface LeaveResponse extends BaseResponse {
  channel_id?: string;
  channel_name?: string;
}

export interface ListChannelsResponse extends BaseResponse {
  channels?: Channel[];
}

export interface MyChannelsResponse extends BaseResponse {
  channels?: Channel[];
}

export interface HistoryResponse extends BaseResponse {
  messages?: (ChatMessage | ChatEvent)[];
  has_more?: boolean;
}

export interface ChannelUsersResponse extends BaseResponse {
  users?: ChannelUser[];
}

export type WebSocketResponse =
  | LoginResponse
  | JoinResponse
  | LeaveResponse
  | ListChannelsResponse
  | MyChannelsResponse
  | HistoryResponse
  | ChannelUsersResponse
  | BaseResponse;

// Event and message types
export interface ChatMessage {
  type: "message";
  channel_id: string;
  user_id: string;
  nickname: string;
  message: string;
  is_passive: boolean;
  sent_at: string;
}

export interface ChatEvent {
  type: "event";
  channel_id?: string;
  event:
    | "joined"
    | "left"
    | "announcement"
    | "nick_change"
    | "kicked"
    | "topic_change";
  user_id?: string;
  nickname?: string;
  sent_at: string;
  message?: string;
  old_nickname?: string;
  new_nickname?: string;
  topic?: string;
  kicked_by?: string;
  reason?: string;
}

export type WebSocketMessage = WebSocketResponse | ChatMessage | ChatEvent;

// Data models
export interface User {
  id: string;
  nickname: string;
  is_serv: boolean;
}

export interface ChannelUser {
  id: number;
  nickname: string;
  is_serv: boolean;
  is_op: boolean;
}

export interface Channel {
  id: string;
  name: string;
  topic: string;
}

export interface Message {
  id: string;
  channel_id: string;
  user_id: string;
  nickname: string;
  message: string;
  is_passive: boolean;
  event: string;
  sent_at: string;
}

export interface Op {
  user_id: string;
  channel_id: string;
  granted_by: string;
  granted_at: string;
}

// WebSocket connection states
export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "error";

// UI state types
export interface AppState {
  connectionState: ConnectionState;
  currentUser: User | null;
  currentChannel: string | null;
  channels: Record<string, Channel>;
  messages: Record<string, Message[]>;
  channelUsers: Record<string, ChannelUser[]>; // channel_id -> ChannelUser[]
  ops: Record<string, string[]>; // channel_id -> user_ids[]
}
