package e2e

import (
	"testing"
)

func TestUserRegistration(t *testing.T) {
	// t.Parallel()
	//
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	//
	// service := stubService(ctx, t)
	//
	// var handler http.Handler
	// handler = rest.New(service, service.Auth, zerolog.Nop())
	// handler = hlog.NewHandler(log.Logger)(handler)
	//
	// srv := httptest.NewServer(handler)
	// defer srv.Close()
	//
	// client := client.New("user")
	// client.BaseURL = srv.URL
	//
	// fakeOauthToken := "fake oauth token"
	// fbUser := eventdb.User{
	// 	FacebookID: "fake fb id",
	// }
	// service.UserFetcher = StubUserFetcher{
	// 	User:  fbUser,
	// 	Token: fakeOauthToken,
	// }
	//
	// client.JWT = loginReply.JWT
	//
	// user, err := client.Users.Get(ctx, "me")
	// if err != nil {
	// 	t.Fatal("GetUser: ", err)
	// }
	// if got, want := loginReply.UserID, user.ID; got != want {
	// 	t.Fatalf("login returned UserID = %q, want %q", got, want)
	// }
	//
	// if _, err := client.Users.List(ctx); err == nil {
	// 	t.Fatalf("non-admin user called Users.List(), should be forbidden")
	// }
	//
	// newZone := "America/New_York"
	// updated, err = client.Users.Update(ctx, "me", eventdb.UserUpdate{
	// 	TimeZone: newZone,
	// })
	// if err != nil {
	// 	t.Fatalf("update time zone: %v", err)
	// }
	// if got, want := updated.TimeZone, newZone; got != want {
	// 	t.Fatalf("updated user TimeZone = %q, want %q", got, want)
	// }
}
