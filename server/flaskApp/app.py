import os
from flask import Flask
from flask_socketio import SocketIO
from fileManager import load_json, save_json

socketio = SocketIO(async_handlers = True)
is_live = os.environ.get("IS_LIVE", "false").lower() == "true"


def create_app() -> Flask:
    """Create an application."""
    # We need to init the config first so all the required files are created
    from config import Config
    from fileManager import ensure_required_directories_exist

    config = Config()
    ensure_required_directories_exist()

    move_anime_mapping_file(config)

    app = create_flask_app()
    app = register_blueprints(app)
    add_frontend_serving_routes(app)

    return app


def add_frontend_serving_routes(app: Flask):
    from flask import render_template, send_file

    # Serve the frontend
    @app.route('/', defaults = {'path': ''})
    @app.route('/<path:path>')
    def index(path):
        return render_template("index.html")

    @app.route('/static/<folder>/<file>')
    def data(folder: str, file: str):
        return send_file(os.path.join('static/static/', folder, file))


def create_flask_app() -> Flask:
    from flask_cors import CORS

    app = Flask(__name__, static_folder = "static/", template_folder = "static")
    CORS(app)
    # Init socketio to allow websockets
    socketio.init_app(app, cors_allowed_origins = "*", async_mode = 'gevent_uwsgi' if is_live else 'gevent')

    return app


def register_blueprints(app: Flask) -> Flask:
    from flaskApp.routes.anilist import anilist_route
    from flaskApp.routes.plex import plex_route
    from flaskApp.routes.config import config_route
    from flaskApp.routes.scheduler import scheduler_route

    app.register_blueprint(plex_route)
    app.register_blueprint(anilist_route)
    app.register_blueprint(scheduler_route)
    app.register_blueprint(config_route)

    return app


def move_anime_mapping_file(config):
    # Move the anime-mapping file if it's live
    if is_live:
        if os.path.exists("data/mapping/anime-mapping.json"):
            mapping_data = load_json("data/mapping/anime-mapping.json")
            save_json(os.path.join(config._MAPPING_PATH, "anime-mapping.json"), mapping_data)

    return is_live
