package main

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/maxGitham77/vueapi/internal/data"
	"net/http"
	"strconv"
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

	// Make sure user is active
	if user.Active == 0 {
		app.errorJSON(w, errors.New("User is not actice"))
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
		Data:    envelop{"token": token, "user": user},
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

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	var users data.User
	all, err := users.GetAll()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	paylaod := jsonResponse{
		Error:   false,
		Message: "success",
		Data:    envelop{"users": all},
	}

	app.writeJSON(w, http.StatusOK, paylaod)
}

func (app *application) EditUser(w http.ResponseWriter, r *http.Request) {
	var user data.User
	err := app.readJSON(w, r, &user)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if user.ID == 0 {
		// Add user
		if _, err := app.models.User.Insert(user); err != nil {
			app.errorJSON(w, err)
			return
		} else {
			// Editing user
			u, err := app.models.User.GetById(user.ID)
			if err != nil {
				app.errorJSON(w, err)
				return
			}

			u.Email = user.Email
			u.Firstname = user.Firstname
			u.Lastname = user.Lastname
			u.Active = user.Active

			if err := u.Update(); err != nil {
				app.errorJSON(w, err)
				return
			}

			// If password != string, update password
			if user.Password != "" {
				err := u.ResetPassword(user.Password)
				if err != nil {
					app.errorJSON(w, err)
					return
				}
			}
		}

		payload := jsonResponse{
			Error:   false,
			Message: "Changes saved",
		}

		_ = app.writeJSON(w, http.StatusAccepted, payload)
	}
}

func (app *application) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	user, err := app.models.User.GetById(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, user)
}

func (app *application) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	err = app.models.User.DeleteByID(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "User deleted",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) LogUserOutAndSetInactive(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	user, err := app.models.User.GetById(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	user.Active = 0
	err = user.Update()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Delete tokens for user
	err = app.models.Token.DeleteTokensForUser(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "User logged out and set to inactive",
	}
	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *application) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Token string `json:"token"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	valid := false
	valid, _ = app.models.Token.ValidToken(requestPayload.Token)

	payload := jsonResponse{
		Error: false,
		Data:  valid,
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) AllBooks(w http.ResponseWriter, r *http.Request) {
	books, err := app.models.Book.GetAll()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "success",
		Data:    envelop{"books": books},
	}

	app.writeJSON(w, http.StatusOK, payload)
}
