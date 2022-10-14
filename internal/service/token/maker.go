package token

import (
	"fmt"
	"time"
)

type Maker interface {
	CreateToken(username string, duration time.Duration) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}

var makerMap = map[string]func(string) (Maker, error){
	"paseto": NewPasetoMaker,
	"jwt":    NewJWTMaker,
}

func NewTokenMaker(engine string, key string) (Maker, error) {
	makerCreator, ok := makerMap[engine]
	if !ok {
		return nil, fmt.Errorf(`token engine "%s" does not exist; use "jwt/paseto"`, engine)
	}

	maker, err := makerCreator(key)
	if err != nil {
		return nil, err
	}

	return maker, nil
}
