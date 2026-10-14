## 1. Regression tests

- [x] 1.1 Add tests in `whatsapp-mcp-server/tests/test_sender_names.py`, with fictitious data only, for each scenario of the `sender-names` spec: direct chat, sender who created a group (in the direct chat and in the group), sender with no direct chat who created a group, number contained in another number, chat named by its number, and own message shown as "Me". Verify that the group and substring cases fail against the current code (`cd whatsapp-mcp-server && uv run pytest tests/test_sender_names.py`).

## 2. Fix

- [x] 2.1 Rewrite `get_sender_name` in `whatsapp-mcp-server/whatsapp.py` to look up only the exact identities (the stored value, and `<number>@s.whatsapp.net` for a bare number), skip non-person JIDs, ignore names equal to the number, and fall back to the number, with no `LIKE` query. Verify that `tests/test_sender_names.py` passes.
- [x] 2.2 Run the whole MCP server suite (`cd whatsapp-mcp-server && uv run pytest`) and verify it passes.

## 3. Live check and docs

- [x] 3.1 After the user starts a new MCP session, read the user's direct chat that showed the wrong sender and the group it came from with `list_messages`, and verify that the direct chat now shows the person's name and the group shows each member correctly.
- [x] 3.2 Validate the change (`openspec validate fix-sender-name-lookup --strict`) and leave the CHANGELOG entry to the user's `/changelogger`.
