package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"
)

type state struct {
	money      int
	betAmount  int
	dealerHand []card
	playerHand []card
	deck       []card
	handCount  int
	username   string
	from       string
}

type card struct {
	suit string
	face string
}

func (c card) String() string {
	return fmt.Sprintf("%s", c.face)
}

// func (c []card) String() string {
// 	return fmt.Sprintf("%s", c.face)
// }

var (
	// TODO: cache it locally??
	states         = map[string]*state{}
	cardFaceValues = map[string]int{
		"A":  0, // handled in a function...
		"2":  2,
		"3":  3,
		"4":  4,
		"5":  5,
		"6":  6,
		"7":  7,
		"8":  8,
		"9":  9,
		"10": 10,
		"J":  10,
		"Q":  10,
		"K":  10,
	}
	suits = []string{
		"hearts",
		"diamonds",
		"spades",
		"clubs",
	}
)

func createShuffledDeck() []card {
	deck := make([]card, 52)
	var deckIndex int
	for _, suit := range suits {
		for face := range cardFaceValues {
			deck[deckIndex] = newCard(suit, face)
			deckIndex++
		}
	}
	// shuffle
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	return deck
}

func newCard(suit string, face string) card {
	return card{
		suit: suit,
		face: face,
	}
}

func newState() *state {
	s := &state{
		money:     100,
		betAmount: 20,
		deck:      createShuffledDeck(),
	}
	s.dealNewHands()
	return s
}

func (s *state) dealNewHands() bool {
	var newShuffle bool
	s.handCount++
	if s.handCount > 4 {
		newShuffle = true
		s.deck = createShuffledDeck()
		s.handCount = 1
	}

	s.dealerHand = make([]card, 0, 10)
	s.playerHand = make([]card, 0, 10)
	s.drawPlayer()
	s.drawDealer()
	s.drawPlayer()
	s.drawDealer()

	return newShuffle
}

func (s *state) drawDealer() {
	var c card
	// TODO mutex!
	if len(s.deck) == 0 {
		return
	}
	c, s.deck = s.deck[len(s.deck)-1], s.deck[:len(s.deck)-1]
	s.dealerHand = append(s.dealerHand, c)
}

func (s *state) drawPlayer() {
	var c card

	// TODO mutex
	if len(s.deck) == 0 {
		return
	}
	c, s.deck = s.deck[len(s.deck)-1], s.deck[:len(s.deck)-1]
	s.playerHand = append(s.playerHand, c)
}

func main() {
	http.HandleFunc("/sms", smsEndpoint)
	fmt.Println("listening on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

var s *state

func smsEndpoint(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic", r)
			w.Write([]byte(lazyTwimlMessage("internal error")))
		}
	}()

	textMessage := r.URL.Query().Get("Body")
	textMessageLower := strings.ToLower(textMessage)
	from := r.URL.Query().Get("From")
	fmt.Printf("%s - request body: %q\n", from, textMessage)

	if strings.HasPrefix(textMessageLower, "list") || strings.HasPrefix(textMessageLower, "scores") {
		statesSlice := make([]*state, 0, len(states))
		for number, state := range states {
			if number == "" {
				continue
			}
			statesSlice = append(statesSlice, state)
		}

		sort.Slice(statesSlice, func(i, j int) bool {
			a := statesSlice[i]
			b := statesSlice[j]
			return a.money > b.money
		})

		scores := make([]string, 0, len(statesSlice))
		for _, s := range statesSlice {
			username := s.username
			if username == "" {
				username = s.from
			}
			scores = append(scores, fmt.Sprintf("$%d: %s", s.money, username))
		}
		message := strings.Join(scores, "\n")

		w.Write([]byte(lazyTwimlMessage(message)))
		return
	}

	s, ok := states[from]
	if !ok {
		// mutex states
		s = newState()
		s.from = from
		states[from] = s

		message := "Welcome to Blackjack!\nYou can type \"my name is {name}\" to set your username\nShuffling Cards...\n\n"
		if s.setUsername(textMessage) {
			message = fmt.Sprintf("%sUsername updated to %s\n\n", message, s.username)
		}
		message = message + s.getStatus()
		w.Write([]byte(lazyTwimlMessage(message)))
		return
	}

	if len(s.deck) == 0 {
		s.dealNewHands()
		message := "no more cards. this should never happen... shuffling"
		message = message + "\n\nDealing new hands\n" + s.getStatus()
		w.Write([]byte(lazyTwimlMessage(message)))
		return
	}

	if s.setUsername(textMessage) {
		w.Write([]byte(lazyTwimlMessage(fmt.Sprintf("Username updated to %s\n\n%s", s.username, s.getStatus()))))
		return
	}

	if strings.HasPrefix(textMessageLower, "reset") {
		s.money = 100
		s.dealNewHands()
		w.Write([]byte(lazyTwimlMessage(fmt.Sprintf("You went bankrupt and you came back to the casino\n\n%s", s.getStatus()))))
		return
	}

	// if strings.HasPrefix(textMessageLower, "max bet") {
	//
	// }

	if strings.HasPrefix(textMessageLower, "hit") || textMessageLower == "h" {
		s.drawPlayer()

		playerHandScore, _ := getHandScore(s.playerHand)
		if playerHandScore > 21 {
			message := fmt.Sprintf("You bust with %d: %s\n", playerHandScore, formatHand(s.playerHand))
			wasShuffled := s.dealNewHands()
			if wasShuffled {
				message = message + "\n\nShuffling deck\n"
			}
			// TODO: can this be in a separate message??
			message = message + "\n\nDealing new hands\n" + s.getStatus()

			w.Write([]byte(lazyTwimlMessage(message)))
			return
		}
		w.Write([]byte(lazyTwimlMessage(s.getStatus())))
	} else if strings.HasPrefix(textMessageLower, "stay") || textMessageLower == "s" {

		s.dealerPlay()

		message := fmt.Sprintf(s.getStayEndStatus())
		wasShuffled := s.dealNewHands()
		if wasShuffled {
			message = message + "\n\nShuffling deck\n"
		}
		// TODO: can this be in a separate message??
		message = message + "\n\nDealing new hands\n" + s.getStatus()
		w.Write([]byte(lazyTwimlMessage(message)))
		return
	} else {
		w.Write([]byte(lazyTwimlMessage("Do you want to hit or stay?")))
		return
	}
	return
}

func (s *state) getStatus() string {
	if len(s.dealerHand) < 2 {
		// TODO: new state
		return "internal error"
	}

	playerHandScore, softScore := getHandScore(s.playerHand)
	score := fmt.Sprintf("%d", playerHandScore)
	if playerHandScore != softScore {
		score = fmt.Sprintf("%d or %d", playerHandScore, softScore)
	}
	return fmt.Sprintf("Dealer's face up card: %s\nYour hand: %v\nYour current score: %s\nType: hit or stay", s.dealerHand[0], formatHand(s.playerHand), score)
}

func (s *state) getStayEndStatus() string {
	dealerScore, _ := getHandScore(s.dealerHand)
	playerScore, _ := getHandScore(s.playerHand)

	winMessage := "Push. You tied with the dealer."
	if dealerScore > 21 {
		s.money = s.money + s.betAmount
		winMessage = fmt.Sprintf("Dealer bust. You won $%d! You now have $%d.", s.betAmount, s.money)
	} else if playerScore > dealerScore {
		s.money = s.money + s.betAmount
		winMessage = fmt.Sprintf("You won $%d! You now have $%d.", s.betAmount, s.money)
	} else if dealerScore > playerScore {
		s.money = s.money - s.betAmount
		winMessage = fmt.Sprintf("You lost $%d. You now have $%d.", s.betAmount, s.money)
	}
	winMessage = fmt.Sprintf("%s\nYour score was %d and the dealer's score was %d.", winMessage, playerScore, dealerScore)
	return fmt.Sprintf("Dealer's hand: %v\nYour hand: %v\n%s\n", formatHand(s.dealerHand), formatHand(s.playerHand), winMessage)
}

func (s *state) dealerPlay() {
	for {
		score, softScore := getHandScore(s.dealerHand)
		fmt.Printf("dealer score: %d or %d\n", score, softScore)

		if score < 17 {
			s.drawDealer()
		} else {
			return
		}
	}
}

func formatHand(cards []card) string {
	s := make([]string, 0, len(cards))
	for _, c := range cards {
		s = append(s, c.face)
	}
	return strings.Join(s, ", ")
}

func doubleBet() {}
func resetBet()  {}
func maxBet()    {}

func getHandScore(cards []card) (score int, softScore int) {
	// add non-aces
	for _, c := range cards {
		if c.face == "A" {
			continue
		}
		score = score + cardFaceValues[c.face]
	}
	softScore = score

	// add aces
	var aceCount int
	for _, c := range cards {
		if c.face != "A" {
			continue
		}

		aceCount++
		score = score + 11
		softScore = softScore + 1
	}

	// convert ace 11s to 1s
	for i := 0; i < aceCount; i++ {
		if score > 21 {
			score = score - 10
		}
	}
	return score, softScore
}

func (s *state) setUsername(textMessage string) bool {
	namePrefix := "my name is "
	textMessageLower := strings.ToLower(textMessage)
	if strings.HasPrefix(textMessageLower, namePrefix) {
		var username string
		if len(textMessageLower)-len(namePrefix) > 32 {
			username = textMessage[len(namePrefix) : 32+len(namePrefix)]
		} else {
			username = textMessage[len(namePrefix):]
		}
		s.username = username
		fmt.Printf("%s set their username to %s", s.from, s.username)
		return true
	}
	return false
}

func lazyTwimlMessage(message string) string {

	// TODO: resp should have remaining money, hands played, last winning
	// split?????

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Message>
        %s
    </Message>
</Response>`, message)
}
