package main

import (
	"net/http"
)

type jsonResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

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
	// Send back a respones
	payload.Error = false
	payload.Message = "Signed in"

	//out, err := json.MarshalIndent(payload, "", "\t")
	err = app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		app.errorLog.Println(err)
	}

	/*w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)*/
}
