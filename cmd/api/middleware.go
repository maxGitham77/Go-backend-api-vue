package main

import "net/http"

func (app *application) AuthTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, err := app.models.Token.AuthenticateToken(request)
		if err != nil {
			payload := jsonResponse{
				Error:   true,
				Message: "Invalid authentication credentials",
			}

			_ = app.writeJSON(writer, http.StatusUnauthorized, payload)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
