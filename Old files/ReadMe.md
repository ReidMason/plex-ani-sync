# Plex-ani-sync

A program to sync Anilist with a plex library so that you no longer need to update your anime list manually.

## Installation

This program is designed to be installed on Unraid through DockerHub, but there are some steps to set it up.

### Getting an Anilist api token
You'll need an Anilist token, get this by following the steps below.
- Sign in to Anilist and go to https://anilist.co/settings/developer
- Create a new api client
- Call it whatever you'd like and set the redirect URL to `https://anilist.co/api/v2/oauth/pin`
- Now edit the url below replacing `{clientId}` with the client id of the API client you just created \
  https://anilist.co/api/v2/oauth/authorize?client_id={clientId}&response_type=token
- Click "Authorize" and copy the key from the text box that appears
- This is your Anilist api key so hold on to it


### Finding your Plex token
You can use your Plex username and password but I recommend you to use a Plex token instead as it's faster.
- Open Plex in your web browser
- Select any show and click the three dots in the bottom left
- Then click `Get info` at the bottom of the menu
- A modal will open, click the text at the bottom left saying `View XML`
- It should open a new page. At the very end of the url there will be your plex token in a url parameter

## Envionment variables explanation
| Variable             | Example                  | Usage                                                                                    |
| -------------------- | ------------------------ | ---------------------------------------------------------------------------------------- |
| PLEX_USERNAME        | SkippyTheSnake           | Username used to log into your Plex account                                              |
| PLEX_PASSWORD        | Pa$$w0rd                 | Password used to log into your Plex account                                              |
| PLEX_SERVER_NAME     | Tower                    | Name of your plex server                                                                 |
| PLEX_SERVER_URL      | http://192.168.0.1:32400 | Url of your plex server (including http://)                                              |
| PLEX_TOKEN           | oijsduh234ok             | Token for accessing Plex server                                                          |
| ANILIST_TOKEN        | kl3456huio45             | Anilist api token                                                                        |
| PLEX_ANIME_LIBRARIES | Anime                    | Name of the library (or list separated by commas) of Plex libraries to look for anime in |
| TIME_TO_RUN          | 21:00                    | Time to run the sync. This is in 24 hour time                                            |

## Adding mappings
All mapping files and persistant files are located in the `/app/data` directory in the container (you should have this mapped to the host system).

Plex provides us with the tvdb ids and the season numbers and we need to be able to map these to Anilist ids. To do this a file called `tvdbid_to_anilistid.json` is used. A mapping will try to be obtained using the anime-offline-database and ScudLee's anime-lists but not all anime are listed on these lists.

When an anime mapping is missing it will be added to a file called `mappingErrors.txt`. Inside this file you will see the series name, season number and the tvdbid of the series.

For basic shows you should then look up the series on Anilist and copy the id from the url which is likely in the format `https://anilist.co/anime/{id}/`. Then open `tvdbid_to_anilistid.json` and search for the series and season you want to map. Inside the season object set the `anilistid` property to a **string** containing the Anilist id.

For example the entry for bersek might look like this.
```json
{
    "tvdbid": "73752",
    "name": "Berserk",
    "ignore": false,
    "seasons": [
        {
            "plex_season_number": "1",
            "plex_additional_seasons": [],
            "anilistid": "33",
            "ignore": false,
            "episode_start": null,
            "season_length": null,
            "unix_last_updated": null
        }
    ]
}
```

For more advanced mappings you can specify specific episode ranges that map to a certain Anilist id, have multiple Plex seasons map to a single Anilist id and ignore seasons or even complete series so they are no longer synched.

One day I'll add some more details on how to set up these complex mappings.

## Sources
Tvdb to anidb mappings obtained from [ScudLee - anime-list](https://github.com/ScudLee/anime-lists)
and [Anime offline database](https://github.com/manami-project/anime-offline-database)
