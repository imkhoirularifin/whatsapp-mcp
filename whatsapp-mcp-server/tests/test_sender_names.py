"""Sender names: exact identity only, never a group or a number that merely contains the digits."""

import sqlite3
from datetime import datetime

import whatsapp
from whatsapp import Message


def _add_chats(stores, chats):
    conn = sqlite3.connect(stores["messages"])
    conn.executemany("INSERT INTO chats VALUES (?, ?, '2026-10-06 00:00:00-06:00')", chats)
    conn.commit()
    conn.close()


def _rename_chat(stores, jid, name):
    conn = sqlite3.connect(stores["messages"])
    conn.execute("UPDATE chats SET name = ? WHERE jid = ?", (name, jid))
    conn.commit()
    conn.close()


def _message(sender, chat_jid, is_from_me=False):
    return Message(
        timestamp=datetime(2026, 10, 6, 12, 0),
        sender=sender,
        content="hola",
        is_from_me=is_from_me,
        chat_jid=chat_jid,
        id="MSGX",
        chat_name="Chat",
    )


def test_sender_with_direct_chat(stores):
    assert whatsapp.get_sender_name("5215533333333") == "Ana López"


def test_sender_who_created_a_group_is_named_by_their_own_chat(stores):
    # The group is stored before the person's chat, so a substring match would find it first
    _add_chats(stores, [
        ("5215599990000-1453593359@g.us", "Familia Díaz"),
        ("5215599990000@s.whatsapp.net", "Luis Díaz"),
    ])
    assert whatsapp.get_sender_name("5215599990000") == "Luis Díaz"
    in_group = whatsapp.format_message(_message("5215599990000", "5215599990000-1453593359@g.us"))
    in_direct_chat = whatsapp.format_message(_message("5215599990000", "5215599990000@s.whatsapp.net"))
    assert "From: Luis Díaz:" in in_group
    assert "From: Luis Díaz:" in in_direct_chat
    assert "Familia Díaz" not in in_direct_chat


def test_sender_without_direct_chat_is_not_named_by_their_group(stores):
    _add_chats(stores, [("5215599999999-1453593359@g.us", "Equipo")])
    assert whatsapp.get_sender_name("5215599999999") == "5215599999999"


def test_number_contained_in_another_number_does_not_match(stores):
    _rename_chat(stores, "5215512345678@s.whatsapp.net", "Rubén García")
    assert whatsapp.get_sender_name("5512345678") == "5512345678"


def test_sender_given_as_jid(stores):
    assert whatsapp.get_sender_name("5215533333333@s.whatsapp.net") == "Ana López"


def test_chat_named_by_its_number_falls_back_to_the_number(stores):
    assert whatsapp.get_sender_name("5215512345678") == "5215512345678"


def test_group_jid_is_never_used_as_a_sender_name(stores):
    assert whatsapp.get_sender_name("120363000000000001@g.us") == "120363000000000001@g.us"


def test_own_message_is_shown_as_me(stores):
    output = whatsapp.format_message(_message("5215500000000", "5215533333333@s.whatsapp.net", is_from_me=True))
    assert "From: Me:" in output
