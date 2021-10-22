from flask import Flask, jsonify, render_template, send_file, request
import os
from flask_cors import CORS

from routes.anilist import anilist_route
from routes.plex import plex_route

app = Flask(__name__, static_folder = "static/", template_folder = "static")
CORS(app)

app.register_blueprint(plex_route)
app.register_blueprint(anilist_route)


@app.route('/', defaults = {'path': ''})
@app.route('/<path:path>')
def index(path):
    return render_template("index.html")


@app.route('/static/<folder>/<file>')
def data(folder: str, file: str):
    return send_file(os.path.join('static/static/', folder, file))


if __name__ == '__main__':
    app.run(host = '0.0.0.0', debug = True)
