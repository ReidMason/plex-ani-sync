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
_Avoid_: Integration, server connection

**Library**:
A named section of a Media Source that groups media of one kind, such as "Anime". Only libraries of shows are relevant.
_Avoid_: Section, collection

**Anime Library**:
A Library a User has chosen as holding anime. Only Anime Libraries are scanned; every Show in one is treated as anime.
_Avoid_: Synced library, enabled library

**Show**:
A series in an Anime Library, made up of Seasons.
_Avoid_: Series, title

**Season**:
A numbered group of Episodes within a Show.

**Episode**:
A single installment of a Season. It is Watched once the User has played it on their Media Source.

**Watched**:
An Episode state: the User's Plex Account has played it at least once. Carries when it was last played.
_Avoid_: Seen, viewed, completed

**Tracker**:
A site a User's watch progress is written to, such as AniList. Added after Media Sources.
_Avoid_: Destination, target, sink

**Entry**:
A single item on a Tracker with its own episode numbering, such as one AniList anime. One Show can span several Entries, and one Entry can span several Seasons.
_Avoid_: Title, listing

**Mapping**:
A link between a run of Episodes in a Show's Season and a run of Episodes in an Entry. A Show is fully mapped when every Episode it has is covered by a Mapping.
_Avoid_: Match, link

**Custom Mapping**:
A Mapping this project maintains by hand for episodes the shared dataset gets wrong or lacks. It wins over any dataset Mapping it overlaps and survives dataset refreshes.
_Avoid_: Correction, override, patch

**Unmapped**:
A Show with no Mapping at all.

**Needs Review**:
A Show whose Mappings exist but disagree with the Episodes it has, such as differing episode counts. It is not synced until the dataset or a Custom Mapping fixes it.
_Avoid_: Broken, invalid
