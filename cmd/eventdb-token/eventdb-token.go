// package main provides a utility for creating service tokens for authenticating with eventdb via firebase
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	firebase "firebase.google.com/go"
	_ "github.com/lib/pq"
	"google.golang.org/api/option"
)

func main() {
	var (
		firebaseProjectID = flag.String("project-id", "the-third-party", "The firebase project-id used for auth")
		serviceAccount    = flag.String("service-account", "", "Google service account JSON file associated with the firebase project")
		apiKey            = flag.String("api-key", "", "A Google API key associate with the firebase projet")
	)
	flag.Parse()

	tokenName := flag.Arg(0)
	if tokenName == "" {
		log.Fatal("usage: eventdb-token <token name>")
	}

	ctx := context.Background()

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: *firebaseProjectID,
	}, option.WithServiceAccountFile(*serviceAccount))
	if err != nil {
		log.Fatal(err)
	}
	auth, err := app.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}

	uid := fmt.Sprintf("service-%s", tokenName)
	customToken, err := auth.CustomToken(uid)
	if err != nil {
		log.Fatal(err)
	}

	verifyReq, err := json.Marshal(map[string]interface{}{
		"returnSecureToken": true,
		"token":             customToken,
	})
	if err != nil {
		log.Fatal(err)
	}

	verifyURL := fmt.Sprintf("https://www.googleapis.com/identitytoolkit/v3/relyingparty/verifyCustomToken?key=%s", *apiKey)
	resp, err := http.Post(verifyURL, "application/json", bytes.NewReader(verifyReq))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		log.Fatal(err)
	}
}
