package main

import (
	"errors"
	"net/http"
	"time"
)

// jsonResponse is the type used for generic JSON response
type jsonResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type envelop map[string]interface{}

// Login is the handler used to attempt to log a user into the api
func (app *application) Login(w http.ResponseWriter, r *http.Request) {
	type credentials struct {
		UserName string `json:"email"`
		Password string `json:"password"`
	}

	var credential credentials
	var payload jsonResponse

	err := app.readJSON(w, r, &credential)
	if err != nil {
		app.errorLog.Println(err)
		payload.Error = true
		payload.Message = "Invalid json supplied, or json missing entirely"
		_ = app.writeJSON(w, http.StatusBadRequest, payload)
	}

	/*err := json.NewDecoder(r.Body).Decode(&credential)
	if err != nil {
		// Send back error message
		app.errorLog.Println("Invalid json")
		payload.Error = true
		payload.Message = "Invalid json"

		out, err := json.MarshalIndent(payload, "", "\t")
		if err != nil {
			app.errorLog.Println(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(out)
		return
	}*/

	// TODO Authenticate
	app.infoLog.Println(credential.UserName, credential.Password)

	// Look up the user by email
	user, err := app.models.User.GetByEmail(credential.UserName)
	if err != nil {
		app.errorJSON(w, errors.New("Invalid username/password"))
		return
	}

	// Validate the user's password
	validPassword, err := user.PasswordMatches(credential.Password)
	if err != nil || !validPassword {
		app.errorJSON(w, errors.New("Invalid username/password"))
		return
	}

	// We have a valid user, so generate a token
	token, err := app.models.Token.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Save it to the database
	err = app.models.Token.Insert(*token, *user)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Send back a respones
	payload = jsonResponse{
		Error:   false,
		Message: "Logged in",
		Data:    envelop{"token": token},
	}

	//out, err := json.MarshalIndent(payload, "", "\t")
	err = app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		app.errorLog.Println(err)
	}

	/*w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)*/
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Token string `json:"token"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, errors.New("Invlid json"))
		return
	}

	err = app.models.Token.DeleteByToken(requestPayload.Token)
	if err != nil {
		app.errorJSON(w, errors.New("Invlid json"))
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "logged out",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}
