import utils
from sync_runner import Syncher

logger = utils.create_logger(__name__)

if __name__ == '__main__':
    logger.info("Starting manual Anilist sync")
    syncher = Syncher()
    syncher.start_sync()
    logger.info("Anilist sync finished")
