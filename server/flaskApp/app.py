from flask import Flask
from flask_socketio import SocketIO

socketio = SocketIO(async_handlers = True)


def create_app(debug: bool = False) -> Flask:
    """Create an application."""
    # We need to init the config first so all the required files are created
    from config import Config
    from fileManager import load_json, save_json
    config = Config()

    import os
    from flask import Flask, render_template, send_file
    from flask_cors import CORS
    from flaskApp.routes.anilist import anilist_route
    from flaskApp.routes.plex import plex_route
    from flaskApp.routes.config import config_route
    from flaskApp.routes.scheduler import scheduler_route

    # Create the required directories
    from fileManager import ensure_required_directories_exist
    ensure_required_directories_exist()

    # Move the anime-mapping file if it's live
    is_live = os.environ.get("IS_LIVE", "false").lower() == "true"
    if is_live:
        if os.path.exists("data/mapping/anime-mapping.json"):
            mapping_data = load_json("data/mapping/anime-mapping.json")
            save_json(os.path.join(config._MAPPING_PATH, "anime-mapping.json"), mapping_data)

    app = Flask(__name__, static_folder = "static/", template_folder = "static")
    CORS(app)
    # socketio = SocketIO(flaskApp, cors_allowed_origins="*")

    app.register_blueprint(plex_route)
    app.register_blueprint(anilist_route)
    app.register_blueprint(scheduler_route)
    app.register_blueprint(config_route)

    # Serve the frontend
    @app.route('/', defaults = {'path': ''})
    @app.route('/<path:path>')
    def index(path):
        return render_template("index.html")

    @app.route('/static/<folder>/<file>')
    def data(folder: str, file: str):
        return send_file(os.path.join('static/static/', folder, file))

    # Init socketio to allow websockets
    socketio.init_app(app, cors_allowed_origins = "*")
    return app
