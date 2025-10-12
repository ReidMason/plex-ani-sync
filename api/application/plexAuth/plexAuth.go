package plexAuth

type PlexPin struct {
	Pin              string
	PinId            int
	ClientIdentifier string
	AppName          string
	AuthToken        *string
}

type PlexPinCreator interface {
	GeneratePin(appName string, clientIdentifier string) (PlexPin, error)
}

type PlexAuth struct {
	PlexPinCreator PlexPinCreator
}

func New(plexPinCreator PlexPinCreator) *PlexAuth {
	return &PlexAuth{
		PlexPinCreator: plexPinCreator,
	}
}
