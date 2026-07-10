package auth

import (
	"context"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type Provider struct {
	oidcProvider *oidc.Provider
	oauthConfig  oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

func NewProvider(ctx context.Context, issuer, clientID, clientSecret, redirectURL string) (*Provider, error) {
	oidcProvider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	oauthConfig := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: clientID})

	return &Provider{
		oidcProvider: oidcProvider,
		oauthConfig:  oauthConfig,
		verifier:     verifier,
	}, nil
}

func (p *Provider) AuthURL(state string) string {
	return p.oauthConfig.AuthCodeURL(state)
}

func (p *Provider) ExternalAuthURL(state, externalIssuer string) string {
	base := p.oauthConfig.AuthCodeURL(state)
	ep := p.oauthConfig.Endpoint.AuthURL
	// Swap the internal host:port for the external one in the auth URL
	for i := 0; i < len(base)-len(ep); i++ {
		if base[i:i+len(ep)] == ep {
			return externalIssuer + "/auth" + base[i+len(ep):]
		}
	}
	return base
}

func (p *Provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.oauthConfig.Exchange(ctx, code)
}

func (p *Provider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return p.verifier.Verify(ctx, rawIDToken)
}

func (p *Provider) UserInfo(ctx context.Context, token oauth2.TokenSource) (*oidc.UserInfo, error) {
	return p.oidcProvider.UserInfo(ctx, token)
}
