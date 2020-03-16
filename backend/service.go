package main

import (
	"log"
	"net/http"
	"time"
	"fmt"
	"strings"
	"encoding/json"
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

type ActiveConnection struct {
	Connection *websocket.Conn
	AuthenticatedUser string
}

var activeConnections = []*ActiveConnection{}

// Inbound from client
type WSCommand struct {
	Command string `json:"command"`
	Params *json.RawMessage `json:"params"`
}

type WSAuthenticateParams struct {
	Token string `json:"token"`
}

type WSSendToTerminalParams struct {
	Text string `json:"text"`
}

type WSTerminalStdoutedBody struct {
	Text string `json:"text"`
}

type WSTerminalStderroredBody struct {
	Text string `json:"text"`
}

// Inform client of something happening server side
type WSAction struct {
	Action string `json:"action"`
	Body interface{} `json:"body"`
}

func processInboundMessage(ac *ActiveConnection, message []byte) {
	var genericCommand WSCommand
	err := json.Unmarshal(message, &genericCommand)
	if err != nil {
		fmt.Println("ERR", err)
	}

	fmt.Println(genericCommand)

	switch genericCommand.Command {
	case "Authenticate":
		var authParams WSAuthenticateParams
		if parseErr := json.Unmarshal(*genericCommand.Params, &authParams); parseErr != nil {
			fmt.Println("Parse error", parseErr)
			return
		}

		fmt.Println(authParams.Token)
		ac.AuthenticatedUser = validateJwt(authParams.Token)
	case "SendToTerminal":
		var termParams WSSendToTerminalParams
		if parseErr := json.Unmarshal(*genericCommand.Params, &termParams); parseErr != nil {
			fmt.Println("Parse error", parseErr)
			return
		}

		fmt.Println(ac.AuthenticatedUser)
		if ac.AuthenticatedUser != "" {
			fmt.Println(termParams.Text)
			serviceContext.InboundMessage <- string(termParams.Text)
		}
	}
}

func connect(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	ac := ActiveConnection{ Connection: c }
	activeConnections = append(activeConnections, &ac)

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		if mt == websocket.TextMessage {
			processInboundMessage(&ac, message)
		}
	}
}

var hmacSampleSecret = []byte("my_secret_key")

func addCorsHeader(res http.ResponseWriter) {
  headers := res.Header()
  headers.Add("Access-Control-Allow-Origin", "*")
  headers.Add("Vary", "Origin")
  headers.Add("Vary", "Access-Control-Request-Method")
  headers.Add("Access-Control-Allow-Credentials", "true")
  headers.Add("Vary", "Access-Control-Request-Headers")
  headers.Add("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, token,Authorization")
  headers.Add("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
}

func auth(w http.ResponseWriter, r *http.Request) {
	addCorsHeader(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
    return
	}

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
		"user": pair[0],
  	"exp": time.Now().UTC().Add(time.Hour * 1).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(hmacSampleSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	fmt.Fprintf(w, tokenString)
}

// Returns username claim if valid or empty string if invalid
func validateJwt(tokenString string) string {
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
		fmt.Println(claims["exp"])
		fmt.Println(claims["user"], claims["user"].(string))

		return claims["user"].(string)
	} else {
		fmt.Println("auth error", err)
		return ""
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
				if c.AuthenticatedUser != "" {
					err := c.Connection.WriteMessage(websocket.TextMessage, []byte(outboundMessage))
					if err != nil {
						log.Println("write err:", err)
					}
				}
			}
		}
	}()

	return serviceContext
}