package plexAuth

type PlexAuthURl string
type PlexToken string
type PlexNonce string
type SignedPlexJwt string

type PlexJwt struct {
	Nonce PlexNonce `json:"nonce"`
	Scope string    `json:"scope"`
	Aud   string    `json:"aud"`
	Iss   string    `json:"iss"`
	Iat   int       `json:"iat"`
	Exp   int       `json:"exp"`
}
