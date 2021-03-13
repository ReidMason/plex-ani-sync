import time

import schedule

from config import TIME_TO_RUN
from sync_runner import Syncher

logger = utils.create_logger(__name__)

def run_sync():
    logger.info("Starting Anilist sync")
    syncher = Syncher()
    syncher.start_sync()
    logger.info("Anilist sync finished")

logger.info("Plex Anilist sync started.")
logger.info(f"Plex Anilist sync waiting for scheduled time ({TIME_TO_RUN}) to run...")
schedule.every().day.at(TIME_TO_RUN).do(run_sync)

while True:
    schedule.run_pending()
    time.sleep(60)
