# sender-names Specification

## Purpose

Defines how the MCP server names the sender of each message it returns, so that every message is attributed to the person who wrote it and never to a group or to someone else.

## Requirements

### Requirement: Sender names come from an exact identity match
When the MCP server returns a message not sent by the user, it SHALL name the sender with the name of the chat stored under exactly that sender's identity: the stored sender value as given, or, when the stored sender is a bare phone number, the JID `<number>@s.whatsapp.net`. It MUST NOT name a sender by partial or substring matching against chat JIDs, and it MUST NOT use the name of a group chat (`@g.us`), a broadcast list or any other JID that does not identify a person.

#### Scenario: Sender with a direct chat
- **WHEN** a message's sender is `5215533333333` and a chat `5215533333333@s.whatsapp.net` is stored with the name "Ana López"
- **THEN** the message is shown as sent by "Ana López"

#### Scenario: Sender who created a group
- **WHEN** a message's sender is `5215533333333`, the group `5215533333333-1453593359@g.us` is stored as "Familia López", and Ana has a direct chat named "Ana López"
- **THEN** the message is shown as sent by "Ana López", never by "Familia López", in the direct chat and in the group

#### Scenario: Sender with no direct chat who created a group
- **WHEN** a message's sender is `5215599999999`, no chat `5215599999999@s.whatsapp.net` exists, and the group `5215599999999-1453593359@g.us` is stored as "Equipo"
- **THEN** the message is shown as sent by `5215599999999`, not by "Equipo"

#### Scenario: Number contained in another number
- **WHEN** a message's sender is `5512345678` and the only stored chat containing those digits is `5215512345678@s.whatsapp.net`, named "Rubén García"
- **THEN** the message is shown as sent by `5512345678`, not by "Rubén García"

### Requirement: Fallback to the sender's number
When no chat is stored under the sender's exact identity, or that chat's stored name is empty or is just the sender's number, the MCP server SHALL show the sender's number. Messages sent by the user SHALL keep being shown as sent by "Me".

#### Scenario: Chat named by its number
- **WHEN** a message's sender is `5215512345678` and its chat `5215512345678@s.whatsapp.net` is stored with the name `5215512345678`
- **THEN** the message is shown as sent by `5215512345678`

#### Scenario: Own message
- **WHEN** a returned message was sent by the user
- **THEN** it is shown as sent by "Me"
