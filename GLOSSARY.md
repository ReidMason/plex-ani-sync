# Plex Ani Sync

Syncs the anime a person watches in a media server to the sites where they track it.

## Language

**User**:
A person who signs in through Authentik. Owns their Media Sources and, later, their Trackers.
_Avoid_: Account, member, profile

**Session**:
A signed-in browser's long-lived, revocable authorisation to act as a User.
_Avoid_: Login, token

**Plex Account**:
A User's plex.tv identity, proven by signing in with Plex. It holds the credential used to talk to any Plex server that account can access. A User links it before adding Plex Media Sources.
_Avoid_: Plex login, Plex user

**Media Source**:
A User's connection to one server that holds their watch history, such as a Plex server at a given address. A User can have several, and a Plex one uses the User's Plex Account.
_Avoid_: Integration, library, server connection

**Tracker**:
A site a User's watch progress is written to, such as AniList. Added after Media Sources.
_Avoid_: Destination, target, sink
