from config import Config
from flask import Blueprint, jsonify, request

from services.animeListServices.anilistService import AnilistService

anilist_route = Blueprint('anilist', __name__, url_prefix = '/api/anilist')


@anilist_route.route('/anilistAuthenticated')
def anilist_authenticated():
    config = Config()

    payload = {'anilistAuthenticated': config.ANILIST_TOKEN is not None}

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

    config = Config()
    config.ANILIST_TOKEN = token
    config.save()

    return jsonify({})


@anilist_route.route('/getPlanningAnimeToRemove')
def get_planning_anime_to_remove():
    config = Config()
    anilist_service = AnilistService(config.ANILIST_TOKEN)
    planning_anime = [x.__dict__ for x in anilist_service.get_planning_anime()]

    return jsonify(planning_anime)


@anilist_route.route('/removePlanningAnime')
def remove_planning_anime():
    config = Config()
    anilist_service = AnilistService(config.ANILIST_TOKEN)
    