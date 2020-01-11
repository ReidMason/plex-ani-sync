import os
import time
import schedule
from config import Config
from syncHandler import start_sync

if __name__ == '__main__':
    sync_time = os.environ.get('sync_time')
    schedule.every().day.at(sync_time).do(lambda: start_sync())
    start_sync()

    while True:
        schedule.run_pending()
        time.sleep(60)
