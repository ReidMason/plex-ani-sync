from datetime import datetime

disallowed_formats = ["MANGA", "NOVEL", "MUSIC", "ONE_SHOT"]


def matching_format(current_format: str, relatives_format: str) -> bool:
    if current_format == relatives_format:
        return True

    tv_formats = ["TV_SHORT", "TV"]
    if current_format in tv_formats and relatives_format in tv_formats:
        return True

    ova_formats = ["OVA", "ONA"]
    if current_format in ova_formats and relatives_format in ova_formats:
        return True

    # Allow an ova to match a sequel as a tv show
    if current_format in ova_formats and relatives_format in tv_formats:
        return True

    return False


class Anime:
    def __init__(self, anime_data: dict) -> None:
        titles = anime_data.get('title', {})
        self.english_title = titles.get('english')
        self.romaji_title = titles.get('romaji')
        self.id: int = anime_data.get('id')
        self.format: str = anime_data.get('format')
        self.episodes: int = anime_data.get('episodes')
        self.synonyms: str = anime_data.get('synonyms')

        self.end_date: datetime = self.extract_date(anime_data.get('endDate'))
        self.start_date: datetime = self.extract_date(anime_data.get('startDate'))

        self.sequel: Anime = None
        self.prequel: Anime = None
        self.supposed_sequel: Anime = None
        self.supposed_prequel: Anime = None
        relation_edges = anime_data.get('relations', {}).get('edges')
        relation_nodes = anime_data.get('relations', {}).get('nodes')
        self.populate_sequel_and_prequel(relation_edges, relation_nodes)

    @property
    def title(self):
        return self.english_title or self.romaji_title

    def populate_sequel_and_prequel(self, relation_edges: dict, relation_nodes: dict) -> None:
        for edge, node in zip(relation_edges or [], relation_nodes or []):
            relation_type = edge.get('relationType')
            media_format = node.get('format')

            sequel_start_date = self.extract_date(node.get('startDate'))
            prequel_end_date = self.extract_date(node.get('endDate'))

            valid_end_date = prequel_end_date is not None and self.start_date is not None
            valid_start_date = sequel_start_date is not None and self.end_date is not None

            if relation_type == "SEQUEL" and media_format not in disallowed_formats and self.supposed_sequel is None and (sequel_start_date > self.end_date if valid_start_date else True):
                self.supposed_sequel = Anime(node)
            elif relation_type == "PREQUEL" and media_format not in disallowed_formats and self.supposed_prequel is None and (prequel_end_date < self.start_date if valid_end_date else True):
                self.supposed_prequel = Anime(node)

            # We don't want anime that is a different type to this anime with exceptions
            if not matching_format(self.format, media_format):
                continue

            # It can't be a sequel unless the start date was after this current animes end date
            if self.sequel is None and relation_type == "SEQUEL" and (sequel_start_date > self.end_date if valid_start_date else True):
                self.sequel = Anime(node)
            # It can't be a prequel unless the end date was before this current animes start date
            elif self.prequel is None and relation_type == "PREQUEL" and (prequel_end_date < self.start_date if valid_end_date else True):
                self.prequel = Anime(node)

            # Sequel and prequel have been filled we can stop looking for them
            if self.sequel is not None and self.prequel is not None:
                break

    def extract_date(self, date_from_request: dict) -> datetime:
        if date_from_request is None:
            return None

        day = date_from_request.get('day')
        month = date_from_request.get('month')
        year = date_from_request.get('year')

        if None in [day, month, year]:
            return None

        return datetime(year, month, day)

    @property
    def season_number(self):
        if self.prequel is None:
            return 1

        return self.prequel.season_number + 1

    @property
    def all_prequels(self):
        prequels_prequels = [self]
        if self.prequel is not None:
            prequels_prequels = self.prequel.all_prequels
            prequels_prequels.append(self)

        return prequels_prequels

    @property
    def all_sequels(self):
        sequels_sequels = [self]
        if self.sequel is not None:
            sequels_sequels = [self] + self.sequel.all_sequels

        return sequels_sequels

    @property
    def all_seasons(self):
        prequels = self.all_prequels
        sequels = self.all_sequels

        return [x for x in prequels if x.id != self.id] + [self] + [x for x in sequels if x.id != self.id]
