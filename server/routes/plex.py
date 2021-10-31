from time import sleep

from config import Config
from flask import Blueprint, json, jsonify, request
from services.plexService import PlexAuthService, PlexService
import itertools

plex_route = Blueprint('plex', __name__, url_prefix='/api/plex')


@plex_route.route('/tokenFilled')
def token_filled():
    config = Config()
    return jsonify({'tokenFilled': config.PLEX_TOKEN != None})


@plex_route.route('/getPin')
def get_pin():
    plex_auth_service = PlexAuthService()
    pin = plex_auth_service.generate_pin()

    return jsonify({'pin': pin})


@plex_route.route('/plexAuthenticated')
def plex_authenticated():
    config = Config()
    plex_service = PlexService(config.PLEX_SERVER_URL, config.PLEX_TOKEN)
    try:
        plex_service.authenticate()
    except Exception:
        return jsonify({'plexAuthenticated': False})

    return jsonify({'plexAuthenticated': True})


@plex_route.route('/getAnime')
def get_anime():
    config = Config()
    plex_service = PlexService(config.PLEX_SERVER_URL, config.PLEX_TOKEN)
    plex_service.authenticate()
    anime = [[y.serialize() for y in x] for x in itertools.islice(plex_service.get_all_anime(), 5)]

    return jsonify(anime)


@plex_route.route('/serverUrlFilled')
def server_url_filled():
    config = Config()
    return jsonify({"serverUrlFilled": config.PLEX_SERVER_URL is not None})


@plex_route.route('/setPlexServerUrl', methods=['POST'])
def set_plex_server_url():
    config = Config()
    data = request.json
    config.PLEX_SERVER_URL = data.get('server_url')
    config.save()
    return jsonify({})
