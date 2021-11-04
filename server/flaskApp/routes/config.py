from config import Config
from flask import Blueprint, json, jsonify

config_route = Blueprint('config', __name__, url_prefix='/api/config')

blacklisted_properties = ['PLEX_SERVER_URL', 'PLEX_TOKEN', 'ANILIST_TOKEN']


@config_route.route('/getConfig')
def get_config():
    config = Config()
    data = config.__dict__
    data = {k: v for k, v in data.items() if k not in blacklisted_properties and not k.startswith("_")}
    return jsonify(data)
