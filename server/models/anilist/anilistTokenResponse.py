class AnilistTokenResponse:
    def __init__(self, response: dict):
        self.token_type = response.get('token_type')
        self.bearer = response.get('Bearer')
        self.expires_in = response.get('expires_in')
        self.access_token = response.get('access_token')
        self.refresh_token = response.get('refresh_token')
