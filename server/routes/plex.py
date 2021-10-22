from time import sleep

from flask import Blueprint, jsonify
from models.configuration import Configuration
from services.plexService import PlexAuthService

plex_route = Blueprint('plex', __name__, url_prefix = '/api/plex')


@plex_route.route('/getPin')
def get_pin():
    plex_auth_service = PlexAuthService()
    pin = plex_auth_service.generate_pin()

    return jsonify({'pin': pin})


@plex_route.route('/plexAuthenticated')
def plex_authenticated():
    config = Configuration()
    payload = {'plexAuthenticated': config.plex_token is not None,
               'token'            : config.plex_token}
    sleep(2)

    return jsonify(payload)
