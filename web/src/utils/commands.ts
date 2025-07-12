import { getters } from "../store";

export interface IRCCommand {
  name: string;
  description: string;
  usage: string;
  minArgs: number;
  maxArgs: number;
  aliases?: string[];
  requiresChannel?: boolean;
  requiresOp?: boolean;
}

export interface ParsedCommand {
  command: string;
  args: string[];
  rawArgs: string;
  isValid: boolean;
  error?: string;
}

export interface CommandSuggestion {
  command: string;
  description: string;
  usage: string;
}

// Define all available IRC commands
export const IRC_COMMANDS: Record<string, IRCCommand> = {
  join: {
    name: "join",
    description: "Join a channel",
    usage: "/join #channel",
    minArgs: 1,
    maxArgs: 1,
    requiresChannel: false,
  },
  leave: {
    name: "leave",
    description: "Leave current channel with optional reason",
    usage: "/leave [reason]",
    minArgs: 0,
    maxArgs: -1, // unlimited
    requiresChannel: true,
  },
  part: {
    name: "part",
    description: "Leave current channel with optional reason",
    usage: "/part [reason]",
    minArgs: 0,
    maxArgs: -1, // unlimited
    aliases: ["leave"],
    requiresChannel: true,
  },
  nick: {
    name: "nick",
    description: "Change your nickname",
    usage: "/nick newname",
    minArgs: 1,
    maxArgs: 1,
    requiresChannel: false,
  },
  me: {
    name: "me",
    description: "Send an action message",
    usage: "/me does something",
    minArgs: 1,
    maxArgs: -1, // unlimited
    requiresChannel: true,
  },
  kick: {
    name: "kick",
    description: "Kick a user from the channel",
    usage: "/kick username [reason]",
    minArgs: 1,
    maxArgs: -1, // unlimited
    requiresChannel: true,
    requiresOp: true,
  },
  topic: {
    name: "topic",
    description: "Change the channel topic",
    usage: "/topic new topic text",
    minArgs: 1,
    maxArgs: -1, // unlimited
    requiresChannel: true,
  },
  announce: {
    name: "announce",
    description: "Make an announcement in the channel (requires op)",
    usage: "/announce announcement text",
    minArgs: 1,
    maxArgs: -1, // unlimited
    requiresChannel: true,
    requiresOp: true,
  },
  help: {
    name: "help",
    description: "Show available commands",
    usage: "/help [command]",
    minArgs: 0,
    maxArgs: 1,
    requiresChannel: false,
  },
};

/**
 * Parse a message to check if it's a command and extract command details
 */
export function parseCommand(message: string): ParsedCommand | null {
  const trimmed = message.trim();
  
  // Not a command if it doesn't start with /
  if (!trimmed.startsWith("/")) {
    return null;
  }

  // Remove leading slash
  const withoutSlash = trimmed.slice(1);
  
  // Split into command and arguments
  const parts = withoutSlash.split(/\s+/);
  const command = parts[0].toLowerCase();
  const args = parts.slice(1);
  const rawArgs = args.join(" ");

  // Check if command exists
  const commandDef = IRC_COMMANDS[command];
  if (!commandDef) {
    return {
      command,
      args,
      rawArgs,
      isValid: false,
      error: `Unknown command: /${command}. Type /help for available commands.`,
    };
  }

  // Validate argument count
  if (args.length < commandDef.minArgs) {
    return {
      command,
      args,
      rawArgs,
      isValid: false,
      error: `Too few arguments. Usage: ${commandDef.usage}`,
    };
  }

  if (commandDef.maxArgs !== -1 && args.length > commandDef.maxArgs) {
    return {
      command,
      args,
      rawArgs,
      isValid: false,
      error: `Too many arguments. Usage: ${commandDef.usage}`,
    };
  }

  // Validate channel requirement
  if (commandDef.requiresChannel && !getters.getCurrentChannelData()) {
    return {
      command,
      args,
      rawArgs,
      isValid: false,
      error: "This command requires you to be in a channel.",
    };
  }

  // Note: Op requirement validation would need access to current user's op status
  // This is handled during execution, not parsing

  return {
    command,
    args,
    rawArgs,
    isValid: true,
  };
}

/**
 * Get command suggestions based on partial input
 */
export function getCommandSuggestions(partialCommand: string): CommandSuggestion[] {
  if (!partialCommand.startsWith("/")) {
    return [];
  }

  const commandPart = partialCommand.slice(1).toLowerCase();
  
  // If no command typed yet, show all commands
  if (commandPart === "") {
    return Object.values(IRC_COMMANDS).map(cmd => ({
      command: `/${cmd.name}`,
      description: cmd.description,
      usage: cmd.usage,
    }));
  }

  // Filter commands that start with the typed text
  return Object.values(IRC_COMMANDS)
    .filter(cmd => cmd.name.startsWith(commandPart))
    .map(cmd => ({
      command: `/${cmd.name}`,
      description: cmd.description,
      usage: cmd.usage,
    }));
}

/**
 * Validate specific command arguments
 */
export function validateCommandArgs(command: string, args: string[]): string | null {
  switch (command) {
    case "join":
      const channelName = args[0];
      if (!channelName.startsWith("#")) {
        return "Channel name must start with #";
      }
      if (channelName.length < 2) {
        return "Channel name is too short";
      }
      if (!/^#[a-zA-Z0-9_-]+$/.test(channelName)) {
        return "Channel name contains invalid characters. Only letters, numbers, underscores, and hyphens are allowed.";
      }
      break;

    case "nick":
      const nickname = args[0];
      if (nickname.length < 1) {
        return "Nickname cannot be empty";
      }
      if (nickname.length > 32) {
        return "Nickname is too long (max 32 characters)";
      }
      if (!/^[a-zA-Z0-9_-]+$/.test(nickname)) {
        return "Nickname contains invalid characters. Only letters, numbers, underscores, and hyphens are allowed.";
      }
      break;

    case "kick":
      const username = args[0];
      if (username.length < 1) {
        return "Username cannot be empty";
      }
      // Additional validation could check if user exists in channel
      break;

    case "topic":
      if (args.join(" ").length > 500) {
        return "Topic is too long (max 500 characters)";
      }
      break;

    case "announce":
      if (args.join(" ").length > 1000) {
        return "Announcement is too long (max 1000 characters)";
      }
      break;
  }

  return null; // No validation errors
}

/**
 * Get help text for a specific command or all commands
 */
export function getCommandHelp(commandName?: string): string {
  if (commandName) {
    const command = IRC_COMMANDS[commandName.toLowerCase()];
    if (!command) {
      return `Unknown command: ${commandName}`;
    }
    return `${command.usage}\n${command.description}`;
  }

  // Return help for all commands
  const commandList = Object.values(IRC_COMMANDS)
    .map(cmd => `  ${cmd.usage.padEnd(30)} - ${cmd.description}`)
    .join("\n");

  return `Available commands:\n${commandList}\n\nType /help <command> for detailed information about a specific command.`;
}

/**
 * Check if a string looks like it might be starting a command
 */
export function isPartialCommand(text: string): boolean {
  return text.startsWith("/") && text.length > 1;
}