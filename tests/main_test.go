package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/shellhub-io/shellhub/pkg/models"
)

type config struct {
	MongoHost string `envconfig:"mongo_host" default:"mongo"`
	MongoPort int    `envconfig:"mongo_port" default:"27017"`
}

func testAPI(e *httpexpect.Expect) {
	type Login struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	e.POST("/api/login").WithForm(Login{"username", "<bad password>"}).
		Expect().
		Status(http.StatusUnauthorized)

	authUser := e.POST("/api/login").WithForm(Login{"username", "password"}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	authUser.Keys().ContainsOnly("user", "name", "tenant", "email", "token")

	token := authUser.Value("token").String().Raw()
	tenant := authUser.Value("tenant").String().Raw()

	authReq := &models.DeviceAuthRequest{
		Info: &models.DeviceInfo{
			ID:         "id",
			PrettyName: "Pretty name",
			Version:    "test",
		},
		DeviceAuth: &models.DeviceAuth{
			TenantID: tenant,
			Identity: &models.DeviceIdentity{
				MAC: "mac",
			},
			PublicKey: "key",
		},
	}
	_ = authReq
	authDevice := e.POST("/api/devices/auth").WithJSON(Login{"username", "password"}).
		Expect().
		Status(http.StatusOK).
		JSON().Object()
	authDevice.Keys().ContainsOnly("name", "namespace", "token", "uid")
	authDevice.Value("name").Equal("mac")
	authDevice.Value("namespace").Equal("username")
	uid := authDevice.Value("uid").String().Raw()

	getDevice := e.GET(fmt.Sprintf("/api/devices/%s", uid)).
		WithHeader("Authorization", "Bearer "+token).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	device := map[string]interface{}{
		"identity": map[string]string{
			"mac": "mac",
		},
		"info": map[string]string{
			"id":          "id",
			"pretty_name": "Pretty name",
			"version":     "test",
		},
		"name":       "mac",
		"namespace":  "username",
		"public_key": "key",
		"status":     "pending",
		"tenant_id":  "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}
	getDevice.ContainsMap(device)

	listDevices := e.GET("/api/devices").
		WithHeader("Authorization", "Bearer "+token).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	for _, val := range listDevices.Iter() {
		val.Object().ContainsMap(device)
	}
	e.GET(fmt.Sprintf("/internal/auth/token/%s", tenant)).
		Expect().
		Status(http.StatusOK)

	renameReq := map[string]interface{}{
		"name": "newName",
	}

	e.PATCH(fmt.Sprintf("/api/devices/%s", uid)).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("X-Tenant-ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithHeader("X-Username", "username").
		WithJSON(renameReq).
		Expect().
		Status(http.StatusOK)

	e.PATCH(fmt.Sprintf("/api/devices/%s/accepted", uid)).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("X-Tenant-ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithHeader("X-Username", "username").
		Expect().
		Status(http.StatusOK)

	// Test for public session routes
	//set a session uid that exists

	session := map[string]interface{}{
		"username":      "username",
		"device_uid":    uid,
		"uid":           "uid",
		"authenticated": false,
	}
	uid_session := "uid"

	authenticated := map[string]interface{}{
		"authenticated": true,
	}

	sessionAuth := map[string]interface{}{
		"username":      "username",
		"device_uid":    uid,
		"uid":           "uid",
		"authenticated": true,
	}

	e.POST("/internal/sessions").WithJSON(session).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	e.PATCH(fmt.Sprintf("/internal/sessions/%s", uid_session)).
		WithJSON(authenticated).
		Expect().
		Status(http.StatusOK)

	getSession := e.GET(fmt.Sprintf("/api/sessions/%s", uid_session)).
		WithHeader("Authorization", "Bearer "+token).
		Expect().
		Status(http.StatusOK).
		JSON().Object()
	getSession.ContainsMap(sessionAuth)

	array := e.GET("/api/sessions").
		WithHeader("Authorization", "Bearer "+token).
		Expect().
		Status(http.StatusOK).JSON().Array()

	for _, val := range array.Iter() {
		val.Object().ContainsMap(sessionAuth)
	}

	e.POST(fmt.Sprintf("/internal/sessions/%s/finish", uid_session)).
		WithHeader("Authorization", "Bearer"+token).
		WithHeader("X-Tenant-ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithHeader("X-Username", "username").
		Expect().
		Status(http.StatusOK)

	// public tests for stats
	e.GET("/api/stats").
		WithHeader("Authorization", "Bearer "+token).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	e.DELETE(fmt.Sprintf("/api/devices/%s", uid)).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("X-Tenant-ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
		WithHeader("X-Username", "username").
		Expect().
		Status(http.StatusOK)

	//public tests for change username

	/*status_array := []int{http.StatusOK, http.StatusOK, http.StatusConflict, http.StatusForbidden}

	forms_array := []interface{}{
		map[string]interface{}{ // successfull email and username change
			"username":        "newusername",
			"email":           "new@email.com",
			"currentPassword": "",
			"newPassword":     "",
		},
		map[string]interface{}{ // successfull password change
			"username":        "",
			"email":           "",
			"currentPassword": "password",
			"newPassword":     "new_password_hash",
		},
		map[string]interface{}{ //conflict
			"username":        "username2",
			"email":           "new@email.com",
			"currentPassword": "",
			"newPassword":     "",
		},
		map[string]interface{}{ // forbidden
			"username":        "",
			"email":           "",
			"currentPassword": "wrong_password",
			"newPassword":     "new_password",
		},
	}

	for i, v := range forms_array {
		e.PUT("/api/user").
			WithHeader("Authorization", "Bearer "+token).
			WithHeader("X-Tenant-ID", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx").
			WithHeader("X-Username", "username").
			WithJSON(v).
			Expect().
			Status(status_array[i])
	}*/

}

func TestEchoClient(t *testing.T) {

	e := httpexpect.WithConfig(httpexpect.Config{
		// prepend this url to all requests
		BaseURL: "http://api:8080/",

		// use http.Client with a cookie jar and timeout
		Client: &http.Client{
			Jar:     httpexpect.NewJar(),
			Timeout: time.Second * 30,
		},

		// use fatal failures
		Reporter: httpexpect.NewRequireReporter(t),

		// use verbose logging
		Printers: []httpexpect.Printer{
			httpexpect.NewCurlPrinter(t),
			httpexpect.NewDebugPrinter(t, true),
		},
	})
	testAPI(e)

}
