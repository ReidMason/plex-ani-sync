package plexAuth

import (
	"fmt"
	"time"

	"github.com/ReidMason/plex-ani-sync/internal/domain/common"
	"github.com/golang-jwt/jwt/v5"
)

func GetPlexJwt(nonce PlexNonce, currentTime time.Time, privateKey []byte) (SignedPlexJwt, error) {
	var plexJwt PlexJwt = PlexJwt{
		Nonce: nonce,
		Scope: "username",
		Aud:   "plex.tv",
		Iss:   common.ClientIdentifier,
		Iat:   int(currentTime.Unix()),
		Exp:   int(currentTime.Unix()) + 3600,
	}

	jwtString, err := plexJwt.signJWT(privateKey)
	if err != nil {
		return SignedPlexJwt(""), err
	}

	return SignedPlexJwt(jwtString), nil
}

// GetClaims implements jwt.Claims interface
func (j PlexJwt) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(int64(j.Exp), 0)), nil
}

func (j PlexJwt) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(int64(j.Iat), 0)), nil
}

func (j PlexJwt) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (j PlexJwt) GetIssuer() (string, error) {
	return j.Iss, nil
}

func (j PlexJwt) GetSubject() (string, error) {
	return "", nil
}

func (j PlexJwt) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{j.Aud}, nil
}

func (j PlexJwt) signJWT(privateKey []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, j)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}
