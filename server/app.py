# We need to init the config first so all the required files are created
from config import Config
from fileManager import load_json, save_json
config = Config()

import os
from flask import Flask, render_template, send_file
from flask_cors import CORS
from routes.anilist import anilist_route
from routes.plex import plex_route
from routes.scheduler import scheduler_route
from routes.config import config_route

# Create the required directories
from fileManager import ensure_required_directories_exist
ensure_required_directories_exist()

# Move the anime-mapping file if it's live
if os.environ.get("IS_LIVE", "false").lower() == "true":
    if os.path.exists("data/mapping/anime-mapping.json"):
        mapping_data = load_json("data/mapping/anime-mapping.json")
        save_json(os.path.join(config._MAPPING_PATH, "anime-mapping.json"), mapping_data)

app = Flask(__name__, static_folder="static/", template_folder="static")
CORS(app)

app.register_blueprint(plex_route)
app.register_blueprint(anilist_route)
app.register_blueprint(scheduler_route)
app.register_blueprint(config_route)


@app.route('/', defaults={'path': ''})
@app.route('/<path:path>')
def index(path):
    return render_template("index.html")


@app.route('/static/<folder>/<file>')
def data(folder: str, file: str):
    return send_file(os.path.join('static/static/', folder, file))


if __name__ == '__main__':
    app.run(host='0.0.0.0', debug=False)
