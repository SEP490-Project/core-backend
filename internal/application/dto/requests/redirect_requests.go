package requests

// RedirectRequest represents the URL parameters for the redirect endpoint
// The hash is extracted from the URL path parameter
type RedirectRequest struct {
	Hash string `uri:"hash" binding:"required,len=16"` // 16-character affiliate link hash
}
