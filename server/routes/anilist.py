from pprint import pprint

from flask import Blueprint, jsonify, request

from models.configuration import Configuration
from services.animeListServices.anilistService import AnilistAuth, AnilistService

anilist_route = Blueprint('anilist', __name__, url_prefix = '/api/anilist')


# Steps required
# Go to the developer portal: https://anilist.co/settings/developer
# Click "create new client"
# Set the name as "Plex ani sync"
# Set the redirect url as "http://10.128.0.101:5000/api/anilist/coderedirect"

# Enter "client id"
# Enter "client secret"

# Generate the auth link to get the token
# Auth link will look like below:
# https://anilist.co/api/v2/oauth/authorize?client_id={client_id}&response_type=token
# Grab the code from the redirect
# Use the code, client_id, client_secret and redirect_url to create the api token


@anilist_route.route('/anilistAuthenticated')
def anilist_authenticated():
    config = Configuration()

    payload = {'anilistAuthenticated': config.anilist_token is not None,
               'token'               : config.anilist_token}

    return jsonify(payload)


@anilist_route.route('/getCodeAuthUrl', methods = ['POST'])
def get_code_auth_url():
    body = request.json
    client_id = body.get('client_id')

    code_auth_url = f"https://anilist.co/api/v2/oauth/authorize?client_id={client_id}&response_type=token"

    return jsonify({"code_auth_url": code_auth_url})


@anilist_route.route('/setAnilistToken', methods = ['POST'])
def set_anilist_token():
    body = request.json
    token = body.get('token')

    if token is None:
        return jsonify({"error": "No token was provided."})

    config = Configuration()
    config.anilist_token = token
    config.save()

    return jsonify({})
