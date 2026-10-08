package auth

import(
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-that-is-at-least-32-characters-long"

func TestTokenManager_RoundTrip(t *testing.T){
	m := NewTokenManager(testSecret, time.Hour)
	id := uuid.New()

	token, exp, err := m.Generate(id)
	if err != nil{
		t.Fatalf("Generate: %v", err)
	}
	if !exp.After(time.Now()){
		t.Errorf("expiry %v should be in the future", exp)
	}
	got, err :=  m.Parse(token)
	if err != nil{
		t.Fatalf("Parse: %v", err)
	}
	if got != id{
		t.Errorf("Parse returned %v, want %v", got, id)
	}
}

func TestTokenManager_Parse_Rejects(t *testing.T){
	m := NewTokenManager(testSecret, time.Hour)
	id := uuid.New()

	valid, _, _ :=m.Generate(id)
	otherSecret, _, _ := NewTokenManager("a-completely-different-secret-32-chars-min", time.Hour).Generate(id)
	expired, _, _ := NewTokenManager(testSecret, -time.Hour).Generate(id) // negative TTL = already expired

	noAlg, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Issuer: tokenIssuer,
		Subject: id.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	badSubject, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: tokenIssuer,
		Subject: "not-a-uuid",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString([]byte(testSecret))

	tests := []struct{
		name string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-jwt"},
		{"tampered signature", valid[:len(valid)-2] + "xx"},
		{"signed with another secret", otherSecret},
		{"expired", expired},
		{"alg none (forged, unsigned)", noAlg},
		{"subject is not a uuid", badSubject},
	}
	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T){
			if _, err := m.Parse(tt.token); err != ErrInvalidToken {
				t.Errorf("Parse error = %v, want ErrInvalidToken", err)
			}
		})
	}
}