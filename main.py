import time
from config import TIME_TO_RUN
from sync_runner import Syncher

import schedule


def run_sync():
    syncher = Syncher()
    syncher.start_sync()


schedule.every().day.at(TIME_TO_RUN).do(run_sync)

while True:
    schedule.run_pending()
    time.sleep(60)
