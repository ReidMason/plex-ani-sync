import logging

import coloredlogs
import selenium.common
from selenium import webdriver
from selenium.common.exceptions import TimeoutException, WebDriverException
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as ec
from selenium.webdriver.support.wait import WebDriverWait

logger = logging.getLogger(__name__)
coloredlogs.install(level = 'DEBUG', fmt = '%(asctime)s [%(name)s] %(message)s', logger = logger)
logging.getLogger('selenium.webdriver.remote.remote_connection').setLevel(logging.WARNING)


class Driver:
    def __init__(self, headless: bool = True):
        """ Creates chromedriver with options for the object. """
        self.logger = logger
        logger.warning("Starting web driver")
        # Setup chrome options
        chrome_options = webdriver.ChromeOptions()
        if headless:
            chrome_options.add_argument('--headless')
            chrome_options.add_argument('--disable-gpu')
            chrome_options.add_argument('--no-sandbox')

        # Remove unwanted logs
        chrome_options.add_argument("--log-level=3")
        self.driver = webdriver.Chrome(chrome_options = chrome_options)
        logger.debug(f"Web driver started")

    def get(self, url):
        """ Loads a given url with the chromedriver. """
        self.driver.get(url)

    def get_html(self, url: str = None) -> str:
        """ Gets the page source of a given url or the current page. """
        if url is not None:
            self.driver.get(url)

        return self.driver.page_source
