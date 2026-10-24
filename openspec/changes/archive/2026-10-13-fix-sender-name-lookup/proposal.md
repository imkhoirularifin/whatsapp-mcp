## Why

`list_messages` and the other read tools show who wrote each message with `get_sender_name`, which falls back to `WHERE jid LIKE '%<number>%'` when the sender has no chat stored under its exact JID. Messages store the sender as a bare number, so the exact lookup never matches and the substring match picks the first chat whose JID merely contains those digits. A group's JID starts with its creator's number (`<creator>-<timestamp>@g.us`), so every message from that person, even in a direct chat, is attributed to the group's name; and a number that contains another person's digits can match too. The user saw a friend's direct messages labelled with the name of a group he created, which misattributes who said what.

## What Changes

- The sender name is resolved only by exact identity: the stored sender as given and, for a bare number, `<number>@s.whatsapp.net`. No substring matching.
- Group chats, broadcast lists and other non-person JIDs are never used as a sender's name.
- A chat whose stored name is just the number is not treated as a name.
- When no name is found, the sender's number is shown, as today.
- Messages from the user keep showing "Me".

## Capabilities

### New Capabilities
- `sender-names`: how the MCP server names the sender of each message it returns.

### Modified Capabilities

## Impact

- **MCP server (Python)**: `get_sender_name` in `whatsapp-mcp-server/whatsapp.py`, used by `format_message` for every message returned by `list_messages` and `get_message_context`; new regression tests.
- **Users**: senders who were wrongly shown with a group's (or another person's) name are now shown with their own chat name or, when they have no direct chat, with their number. Senders who only appear in groups keep showing their number; naming them from the address book is left for the TODO item on N+1 sender lookups.
- **Bridge, database, REST API**: no change.
