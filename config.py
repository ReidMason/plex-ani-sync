import os


class Config:
    """ A simple class to hold all the configurable variables for the program """

    def __init__(self):
        self.libraries = (os.environ.get('libraries') or '').split()
        self.server_token = os.environ.get('server_token')
        self.server_url = os.environ.get('server_url')
        self.anilist_access_token = os.environ.get('anilist_access_token')
