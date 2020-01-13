import logging
import os
import sys
import time

import coloredlogs
import schedule
from anilist import Anilist
from plexConnection import PlexConnection

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)


def do_sync():
    try:
        from syncHandler import start_sync
        start_sync()

    # These errors can be fixed without restarting the docker container
    except PlexConnection.PlexServerUnreachable as e:
        logger.error(e)

    # These errors can only be fixed by changing data and restarting the docker container so the program should end
    except Anilist.InvalidToken and PlexConnection.InvalidPlexToken as e:
        logger.error(e)
        sys.exit()


if __name__ == '__main__':
    sync_time = os.environ.get('sync_time')
    schedule.every().day.at(sync_time).do(lambda: do_sync())
    do_sync()

    while True:
        schedule.run_pending()
        time.sleep(60)
