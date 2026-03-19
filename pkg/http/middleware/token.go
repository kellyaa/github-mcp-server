package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	ghcontext "github.com/github/github-mcp-server/pkg/context"
	"github.com/github/github-mcp-server/pkg/http/oauth"
	"github.com/github/github-mcp-server/pkg/utils"
)

func ExtractUserToken(oauthCfg *oauth.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Check if token info already exists in context, if it does, skip extraction.
			// In remote setup, we may have already extracted token info earlier.
			if _, ok := ghcontext.GetTokenInfo(ctx); ok {
				// Token info already exists in context, skip extraction
				next.ServeHTTP(w, r)
				return
			}

			tokenType, token, err := utils.ParseAuthorizationHeader(r)
			if err != nil {
				// For missing Authorization header, fall back to GITHUB_PERSONAL_ACCESS_TOKEN env var
				if errors.Is(err, utils.ErrMissingAuthorizationHeader) {
					envToken := os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
					if envToken != "" {
						tokenType, token = utils.ClassifyToken(envToken)
					} else {
						sendAuthChallenge(w, r, oauthCfg)
						return
					}
				} else {
					// For other auth errors (bad format, unsupported), return 400
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}

			ctx = ghcontext.WithTokenInfo(ctx, &ghcontext.TokenInfo{
				Token:     token,
				TokenType: tokenType,
			})
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// sendAuthChallenge sends a 401 Unauthorized response with WWW-Authenticate header
// containing the OAuth protected resource metadata URL as per RFC 6750 and MCP spec.
func sendAuthChallenge(w http.ResponseWriter, r *http.Request, oauthCfg *oauth.Config) {
	resourcePath := oauth.ResolveResourcePath(r, oauthCfg)
	resourceMetadataURL := oauth.BuildResourceMetadataURL(r, oauthCfg, resourcePath)
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata=%q`, resourceMetadataURL))
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
