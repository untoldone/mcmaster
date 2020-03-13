package main

import (
	"log"
	"net/http"
	"time"
	"fmt"
	"strings"
	"encoding/base64"
	"github.com/gorilla/websocket"
	"github.com/dgrijalva/jwt-go"
	"github.com/JoshuaDoes/go-yggdrasil"
)

type Service struct {
	
}

type ServiceContext struct {
	InboundMessage chan string
	OutboundMessage chan string
}

// checkSameOrigin returns true if the origin is not set or is equal to the request host.
func checkSameOrigin(r *http.Request) bool {
	return true
	/*origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}
	return equalASCIIFold(u.Host, r.Host)*/
}

var upgrader = websocket.Upgrader{
	CheckOrigin: checkSameOrigin,
} // use default options

var serviceContext = ServiceContext{
	InboundMessage: make(chan string),
	OutboundMessage: make(chan string),
}

var activeConnections = []*websocket.Conn{}

func connect(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	activeConnections = append(activeConnections, c)
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		if mt == websocket.TextMessage {
			serviceContext.InboundMessage <- string(message)
		}

		/*err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}*/
	}
}

var hmacSampleSecret = []byte("my_secret_key")

func auth(w http.ResponseWriter, r *http.Request) {
	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)

  if len(auth) != 2 || auth[0] != "Basic" {
      http.Error(w, "authorization failed", http.StatusUnauthorized)
      return
  }

	payload, _ := base64.StdEncoding.DecodeString(auth[1])
  pair := strings.SplitN(string(payload), ":", 2)

  if len(pair) != 2 {
      http.Error(w, "authorization failed", http.StatusUnauthorized)
      return
  }

	yggdrasilClient := &yggdrasil.Client{ClientToken: "your client token here"}
	_, yErr := yggdrasilClient.Authenticate(pair[0], pair[1], "Minecraft", 1)
	if yErr != nil {
		http.Error(w, fmt.Sprintf("authorization failed: %s", yErr), http.StatusUnauthorized)
		return
	}

	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
  	"exp": time.Now().UTC().Add(time.Hour * 1).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(hmacSampleSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	fmt.Fprintf(w, tokenString)
}

func validateJwt(tokenString string) bool {
	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
	    // Don't forget to validate the alg is what you expect:
	    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
	        return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
	    }

	    // hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
	    return hmacSampleSecret, nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
	    fmt.Println(claims["foo"], claims["nbf"])
	    return true
	} else {
	    fmt.Println(err)
	    return false
	}
}

func (l *Service) Run(addr string) (ServiceContext) {
	http.HandleFunc("/ws", connect)
	http.HandleFunc("/auth", auth)

	go func(){
		log.Fatal(http.ListenAndServe(addr, nil))
	}()

	go func(){
		for outboundMessage := range serviceContext.OutboundMessage {
			for _, c := range activeConnections {
				err := c.WriteMessage(websocket.TextMessage, []byte(outboundMessage))
				if err != nil {
					log.Println("write err:", err)
				}
			}
		}
	}()

	return serviceContext
}